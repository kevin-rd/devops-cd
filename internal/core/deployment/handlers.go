package deployment

import (
	"context"
	"devops-cd/internal/adapter/deploy"
	"devops-cd/internal/core/release_app"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/crypto"
	"devops-cd/pkg/constants"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

type Handler interface {
	Handle(ctx context.Context, dep *model.Deployment) (status string, updateFunc func(*model.Deployment), err error)
}

type HandlerFunc func(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error)

func (h HandlerFunc) Handle(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	return h(ctx, dep)
}

func (sm *StateMachine) registerHandlers() {
	sm.handlers[constants.DeploymentStatusPending] = HandlerFunc(sm.HandlePending)
	sm.handlers[constants.DeploymentStatusRunning] = HandlerFunc(sm.HandleRunning)
}

// handlers

// HandlePending handle Pending -> Running
func (sm *StateMachine) HandlePending(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	// 1. 触发部署（幂等）
	if err := sm.toDeploy(ctx, dep); err != nil {
		return constants.DeploymentStatusFailed, func(deployment *model.Deployment) {
			setErrorMessage(deployment, err.Error())
			deployment.RetryCount++
		}, nil
	}

	// 2. 变更为 Running
	return constants.DeploymentStatusRunning, func(d *model.Deployment) {
		d.TaskID = dep.TaskID
		setErrorMessage(d, "")
	}, nil
}
func (sm *StateMachine) toDeploy(ctx context.Context, dep *model.Deployment) error {
	var deployment model.Deployment
	if err := sm.db.Where("id = ?", dep.ID).Preload("Application").Preload("Cluster").First(&deployment).Error; err != nil {
		return err
	}

	// 加载 ReleaseApp / Build / ProjectEnvConfig，以解析 artifacts_json（chart 来源、chart name、values 等）
	var rel model.ReleaseApp
	if err := sm.db.First(&rel, deployment.ReleaseID).Error; err != nil {
		return fmt.Errorf("load release_app failed: %w", err)
	}
	if rel.BuildID == nil {
		return fmt.Errorf("release_app.build_id 为空")
	}
	var build model.Build
	if err := sm.db.First(&build, *rel.BuildID).Error; err != nil {
		return fmt.Errorf("load build failed: %w", err)
	}

	// 需要 Project 用于模板变量
	var app model.Application
	if err := sm.db.Preload("Project").First(&app, deployment.AppID).Error; err != nil {
		return fmt.Errorf("load app failed: %w", err)
	}

	var projectCfg model.ProjectEnvConfig
	if err := sm.db.Where("project_id = ? AND env = ?", app.ProjectID, deployment.Env).First(&projectCfg).Error; err != nil {
		return fmt.Errorf("load project_env_config failed: %w", err)
	}

	artifacts, err := release_app.LoadArtifactsV1(&projectCfg)
	if err != nil {
		return err
	}
	if artifacts.AppChart == nil || !artifacts.AppChart.Enabled {
		return fmt.Errorf("app_chart 未启用")
	}

	// appEnvConfig: 仅用于模板变量（cluster/env）
	appEnvCfg := &model.AppEnvConfig{Env: deployment.Env, Cluster: deployment.ClusterName}
	tplCtx := release_app.RenderTemplateContext(&app, &build, appEnvCfg)

	// 1) 若需要，先部署 config_chart
	if artifacts.ConfigChart != nil && artifacts.ConfigChart.Enabled && artifacts.AppChart.DependsOnConfigChart {
		if err := sm.deployOneChart(ctx, &app, &build, deployment, tplCtx, artifacts.ConfigChart, "config_chart"); err != nil {
			return err
		}
	}

	// 2) 部署 app_chart
	return sm.deployOneChart(ctx, &app, &build, deployment, tplCtx, artifacts.AppChart, "app_chart")
}

func (sm *StateMachine) deployOneChart(
	ctx context.Context,
	app *model.Application,
	build *model.Build,
	deployment model.Deployment,
	tplCtx map[string]interface{},
	chartSpec *release_app.ChartSpecV1,
	kind string,
) error {
	if chartSpec == nil || !chartSpec.Enabled {
		return nil
	}

	chartNameTpl := chartSpec.Chart.ChartNameTemplate
	if chartNameTpl == "" {
		chartNameTpl = "{{.app_type}}"
	}
	chartName, err := release_app.ParseTemplateForInternal(chartNameTpl, tplCtx)
	if err != nil {
		return fmt.Errorf("%s: chart_name_template 解析失败: %w", kind, err)
	}

	chartVersion := ""
	if chartSpec.Chart.ChartVersionTemplate != "" {
		chartVersion, err = release_app.ParseTemplateForInternal(chartSpec.Chart.ChartVersionTemplate, tplCtx)
		if err != nil {
			return fmt.Errorf("%s: chart_version_template 解析失败: %w", kind, err)
		}
	}

	artifactURL := ""
	if chartSpec.Chart.ArtifactURLTemplate != "" {
		artifactURL, err = release_app.ParseTemplateForInternal(chartSpec.Chart.ArtifactURLTemplate, tplCtx)
		if err != nil {
			return fmt.Errorf("%s: artifact_url_template 解析失败: %w", kind, err)
		}
	}

	// values[]：kind 为 config_chart 时用其 values；app_chart 的 values 在 release_app 阶段已计算并落库到 deployment.Values
	valuesMap := map[string]interface{}(deployment.Values)
	if kind == "config_chart" {
		// config_chart values 需要在此处计算
		appEnvCfg := &model.AppEnvConfig{Env: deployment.Env, Cluster: deployment.ClusterName}
		m, err := release_app.ParseValuesV1(sm.db, app, build, nil, appEnvCfg, chartSpec.Values)
		if err != nil {
			return fmt.Errorf("%s: values 计算失败: %w", kind, err)
		}
		valuesMap = m
	}

	param := deploy.DeploymentParam{
		AppName: app.Name,
		AppType: app.AppType,
		Values:  valuesMap,

		ReleaseName: deployment.DeploymentName,
		Env:         deployment.Env,
		Namespace:   deployment.Namespace,

		Kubeconfig: deployment.Cluster.Kubeconfig,

		ChartSourceType:  chartSpec.Chart.Type,
		ChartName:        chartName,
		ChartVersion:     chartVersion,
		ChartRepoURL:     chartSpec.Chart.RepoURL,
		ChartArtifactURL: artifactURL,
	}

	// chart repo 认证（v1：仅 basic_auth；credential_ref 支持 "id:123" 或 "123"）
	if chartSpec.Chart.CredentialRef != "" {
		if u, p, err := sm.resolveBasicAuth(chartSpec.Chart.CredentialRef); err == nil {
			param.ChartUsername = u
			param.ChartPassword = p
		}
	}

	// config_chart 的 release_name 可以用 template 覆盖
	if kind == "config_chart" && chartSpec.ReleaseNameTemplate != "" {
		if rn, err := release_app.ParseTemplateForInternal(chartSpec.ReleaseNameTemplate, tplCtx); err == nil && rn != "" {
			param.ReleaseName = rn
		}
	}
	if kind == "config_chart" && param.ReleaseName == deployment.DeploymentName {
		// 没配置模板时，给 config chart 一个默认 release name，避免与 app release 冲突
		param.ReleaseName = fmt.Sprintf("%s-config", deployment.DeploymentName)
	}
	if kind == "app_chart" && chartSpec.ReleaseNameTemplate != "" {
		if rn, err := release_app.ParseTemplateForInternal(chartSpec.ReleaseNameTemplate, tplCtx); err == nil && rn != "" {
			param.ReleaseName = rn
		}
	}

	return sm.deployer.Deploy(ctx, &param)
}

func (sm *StateMachine) resolveBasicAuth(ref string) (string, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", nil
	}
	idStr := ref
	if strings.HasPrefix(ref, "id:") {
		idStr = strings.TrimPrefix(ref, "id:")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return "", "", err
	}
	var c model.Credential
	if err := sm.db.First(&c, id).Error; err != nil {
		return "", "", err
	}
	plain, err := crypto.Decrypt(c.EncryptedData)
	if err != nil {
		return "", "", err
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(plain), &m); err != nil {
		return "", "", err
	}
	// v1: 仅支持 basic_auth 的 username/password
	return m["username"], m["password"], nil
}

// HandleRunning handle Running → Success / Failed
func (sm *StateMachine) HandleRunning(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	log := sm.logger.With(zap.Int64("deployment_id", dep.ID)).Sugar()

	// todo
	status, err := sm.deployer.CheckStatus(ctx, nil)
	if err != nil {
		return "", nil, fmt.Errorf("check status failed: %w", err)
	}

	switch status {
	case "success":
		return constants.DeploymentStatusSuccess, func(d *model.Deployment) {
			now := time.Now()
			d.FinishedAt = &now
			setErrorMessage(d, "")
		}, nil //nolint:staticcheck
	case "failed":
		return constants.DeploymentStatusFailed, func(d *model.Deployment) {
			setErrorMessage(d, "deploy failed")
			d.RetryCount++
		}, nil
	default:
		log.Debugf("[Deployment SM: %d-%d-%d] 部署进行中: status: %v", dep.BatchID, dep.ReleaseID, dep.ID, status)
		return "", nil, nil // 继续等待
	}
}

func setErrorMessage(dep *model.Deployment, msg string) {
	if msg == "" {
		dep.ErrorMessage = nil
		return
	}
	if dep.ErrorMessage == nil {
		dep.ErrorMessage = new(string)
	}
	*dep.ErrorMessage = msg
}
