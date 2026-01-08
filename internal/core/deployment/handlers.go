package deployment

import (
	"context"
	"devops-cd/internal/core/deployment/helpers/tpl"
	"devops-cd/internal/core/deployment/plan/drivers"
	helmDriver "devops-cd/internal/core/deployment/plan/drivers/helm"
	"fmt"
	"strings"
	"time"

	"devops-cd/internal/model"
	"devops-cd/pkg/constants"

	"go.uber.org/zap"
)

// ============ handler definition

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

// ========== all handlers

// HandlePending handle Pending -> Running
func (sm *StateMachine) HandlePending(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	startedAt := time.Now()

	// 1. 执行 pre/main 两阶段（当前按同步闭环执行，避免引入 stage 落库字段）
	namespace, deploymentName, mainDriverType, err := sm.executeStages(ctx, dep.ID)
	if err != nil {
		return constants.DeploymentStatusFailed, func(d *model.Deployment) {
			setErrorMessage(d, err.Error())
			d.RetryCount++
			d.StartedAt = &startedAt
			finishedAt := time.Now()
			d.FinishedAt = &finishedAt
		}, nil
	}

	// 2. pre 已同步完成，main 已触发：进入 Running（FinishedAt 不应在此处写入）
	return constants.DeploymentStatusRunning, func(d *model.Deployment) {
		d.Namespace = namespace
		d.DeploymentName = deploymentName
		mt := mainDriverType
		d.DriverType = &mt
		d.StartedAt = &startedAt
		d.FinishedAt = nil
		setErrorMessage(d, "")
	}, nil
}

// executeStages:
// - pre 阶段（config_chart）同步执行，失败直接返回错误
// - main 阶段（app_chart）触发一次 Deploy，并返回 main driver_type（供 Running 阶段 CheckStatus 使用）
func (sm *StateMachine) executeStages(ctx context.Context, deploymentID int64) (namespace string, deploymentName string, mainDriverType string, err error) {
	var dep model.Deployment
	if err := sm.db.WithContext(ctx).Where("id = ?", deploymentID).Preload("Cluster").First(&dep).Error; err != nil {
		return "", "", "", err
	}

	// 加载 ReleaseApp / Build
	var rel model.ReleaseApp
	if err := sm.db.WithContext(ctx).Preload("Build").First(&rel, dep.ReleaseID).Error; err != nil {
		return "", "", "", fmt.Errorf("load release_app failed: %w", err)
	}
	if rel.Build == nil {
		return "", "", "", fmt.Errorf("load Build failed when load ReleaseApp")
	}

	// Load App / ProjectEnvConfig
	var app model.Application
	if err := sm.db.WithContext(ctx).Preload("Project").Preload("Repository").First(&app, dep.AppID).Error; err != nil {
		return "", "", "", fmt.Errorf("load app failed: %w", err)
	}
	var projectCfg model.ProjectEnvConfig
	if err := sm.db.WithContext(ctx).Where("project_id = ? AND env = ?", app.ProjectID, dep.Env).First(&projectCfg).Error; err != nil {
		return "", "", "", fmt.Errorf("load project_env_config failed: %w", err)
	}

	// repo.app_count：当前 project 下，该 repo 关联的应用数（排除 deleted）
	var repoAppCount int64
	if err := sm.db.WithContext(ctx).
		Model(&model.Application{}).
		Where("project_id = ? AND repo_id = ?", app.ProjectID, app.RepoID).
		Count(&repoAppCount).Error; err != nil {
		return "", "", "", fmt.Errorf("count repo apps failed: %w", err)
	}
	tplOpts := &tpl.ContextOptions{
		Repo:         app.Repository,
		RepoAppCount: &repoAppCount,
	}

	// 解析 artifacts_json
	arts, err := model.LoadArtifactsV1(projectCfg.ArtifactsJSON)
	if err != nil {
		return "", "", "", err
	}
	if arts.AppChart == nil || !arts.AppChart.Enabled {
		return "", "", "", fmt.Errorf("app_chart 未启用")
	}
	if strings.TrimSpace(arts.AppChart.Type) == "" {
		return "", "", "", fmt.Errorf("app_chart.type 为空")
	}
	if arts.ConfigChart != nil && arts.ConfigChart.Enabled && strings.TrimSpace(arts.ConfigChart.Type) == "" {
		return "", "", "", fmt.Errorf("config_chart.type 为空")
	}

	// 1) namespace：由 deployment 层统一计算（driver 外部），并传入各 stage
	nsTpl := strings.TrimSpace(arts.NamespaceTemplate)
	if nsTpl == "" {
		return "", "", "", fmt.Errorf("namespace_template 为空")
	}
	renderCtx := tpl.RenderTemplateContext(&app, rel.Build, dep.Env, dep.ClusterName, tplOpts)
	ns, err := tpl.ParseTemplate(nsTpl, renderCtx)
	if err != nil {
		return "", "", "", fmt.Errorf("namespace_template 解析失败: %w", err)
	}
	if strings.TrimSpace(ns) == "" {
		return "", "", "", fmt.Errorf("namespace_template 解析结果为空")
	}

	helmPayload := &helmDriver.ExecutePayload{
		Deployment: &dep,
		App:        &app,
		Build:      rel.Build,
		ProjectCfg: &projectCfg,
		Artifacts:  arts,
		TplOptions: tplOpts,
	}

	// 2) Pre: config chart
	if arts.ConfigChart != nil && arts.ConfigChart.Enabled {
		dv, ok := sm.registry.Get(arts.ConfigChart.Type)
		if !ok {
			return "", "", "", fmt.Errorf("driver not found: %s", arts.ConfigChart.Type)
		}

		if _, err = dv.Execute(ctx, &drivers.ExecuteRequest{Stage: drivers.StagePre, Namespace: ns, Payload: helmPayload}); err != nil {
			return ns, "", "", err
		}
	}
	// 3) Main: app chart
	mainType := strings.TrimSpace(arts.AppChart.Type)
	dv, ok := sm.registry.Get(mainType)
	if !ok {
		return "", "", "", fmt.Errorf("driver not found: %s", mainType)
	}
	if _, err = dv.Execute(ctx, &drivers.ExecuteRequest{Stage: drivers.StageMain, Namespace: ns, Payload: helmPayload}); err != nil {
		return ns, "", mainType, err
	}

	// main 的 deployment_name：由 deployment 层根据 app_chart.data.release_name_template 计算并回填
	// 当前先复用 helm driver 的 config 解析（因为 driver_type=helm）
	deploymentName = app.Name
	if mainType == "helm" && arts.AppChart != nil {
		if cfg, err2 := helmDriver.DecodeConfig(arts.AppChart.Data); err2 == nil && strings.TrimSpace(cfg.ReleaseNameTemplate) != "" {
			if dn, err3 := tpl.ParseTemplate(cfg.ReleaseNameTemplate, renderCtx); err3 == nil && strings.TrimSpace(dn) != "" {
				deploymentName = dn
			}
		}
	}
	return ns, deploymentName, mainType, nil
}

// HandleRunning handle Running → Success / Failed
func (sm *StateMachine) HandleRunning(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	log := sm.logger.With(zap.Int64("deployment_id", dep.ID)).Sugar()

	// 重新加载 deployment + cluster（用于获取 kubeconfig 做状态检查）
	var full model.Deployment
	if err := sm.db.WithContext(ctx).Where("id = ?", dep.ID).Preload("Cluster").First(&full).Error; err != nil {
		return "", nil, err
	}

	// main 阶段状态检查：根据 dep.driver_type 选择 driver
	if full.DriverType == nil || strings.TrimSpace(*full.DriverType) == "" {
		return constants.DeploymentStatusFailed, func(d *model.Deployment) {
			setErrorMessage(d, "driver_type 为空，无法检查部署状态")
			now := time.Now()
			d.FinishedAt = &now
		}, nil
	}
	driverType := strings.TrimSpace(*full.DriverType)
	dv, ok := sm.registry.Get(driverType)
	if !ok {
		return constants.DeploymentStatusFailed, func(d *model.Deployment) {
			setErrorMessage(d, fmt.Sprintf("driver not found: %s", driverType))
			now := time.Now()
			d.FinishedAt = &now
		}, nil
	}

	// helm 不使用 task_id，直接用当前 deployment 字段（namespace/deployment_name）
	res, err := dv.CheckStatus(ctx, &drivers.ExecuteRequest{
		Stage:     drivers.StageMain,
		Namespace: full.Namespace,
		Payload:   &full, // driver 可按需断言使用
	})
	if err != nil {
		return "", nil, fmt.Errorf("check status failed: %w", err)
	}

	switch res.Status {
	case drivers.StatusSuccess:
		return constants.DeploymentStatusSuccess, func(d *model.Deployment) {
			now := time.Now()
			d.FinishedAt = &now
			setErrorMessage(d, "")
		}, nil
	case drivers.StatusFailed:
		return constants.DeploymentStatusFailed, func(d *model.Deployment) {
			setErrorMessage(d, res.Message)
			d.RetryCount++
			now := time.Now()
			d.FinishedAt = &now
		}, nil
	default:
		log.Debugf("[Deployment SM: %d-%d-%d] 部署进行中: status: %v", full.BatchID, full.ReleaseID, full.ID, res.Status)
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
