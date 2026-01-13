package helm

import (
	"context"
	"devops-cd/internal/core/deployment/helpers/tpl"
	"devops-cd/internal/core/deployment/plan/drivers"
	"devops-cd/internal/pkg/logger"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"devops-cd/internal/model"
	"devops-cd/internal/pkg/crypto"

	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

type ExecutePayload struct {
	Deployment *model.Deployment
	App        *model.Application
	Build      *model.Build
	ProjectCfg *model.ProjectEnvConfig
	Artifacts  *model.ArtifactsV1
	TplOptions *tpl.ContextOptions
}

type Driver struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Driver {
	return &Driver{db: db}
}

func (d *Driver) Name() string { return "helm" }

func (d *Driver) Execute(ctx context.Context, req *drivers.ExecuteRequest) (*drivers.ExecuteResult, error) {
	p, ok := req.Payload.(*ExecutePayload)
	if !ok || p == nil || p.Deployment == nil || p.App == nil || p.Build == nil || p.Artifacts == nil {
		return nil, fmt.Errorf("helm driver: invalid payload")
	}

	switch req.Stage {
	case drivers.StagePre:
		return d.execChart(ctx, req.Namespace, p, p.Artifacts.ConfigChart, "config_chart")
	case drivers.StageMain:
		return d.execChart(ctx, req.Namespace, p, p.Artifacts.AppChart, "app_chart")
	default:
		return nil, fmt.Errorf("helm driver: unknown stage: %s", req.Stage)
	}
}

func (d *Driver) CheckStatus(ctx context.Context, req *drivers.ExecuteRequest) (*drivers.ExecuteResult, error) {
	// helm：基于 namespace + releaseName 查询 release 状态
	dep, ok := req.Payload.(*model.Deployment)
	if !ok || dep == nil {
		// 允许直接传 Deployment
		if v, ok2 := req.Payload.(model.Deployment); ok2 {
			dep = &v
		}
	}
	if dep == nil {
		return nil, fmt.Errorf("helm CheckStatus: payload must be deployment")
	}
	if strings.TrimSpace(dep.Namespace) == "" || strings.TrimSpace(dep.DeploymentName) == "" {
		return nil, fmt.Errorf("helm CheckStatus: namespace/deployment_name 为空")
	}
	if dep.Cluster == nil || strings.TrimSpace(dep.Cluster.Kubeconfig) == "" {
		return nil, fmt.Errorf("helm CheckStatus: cluster/kubeconfig 为空（需要 Preload Cluster）")
	}

	restClientGetter, err := NewRESTClientGetter(dep.Cluster.Kubeconfig, dep.Namespace)
	if err != nil {
		return nil, err
	}
	actionConfig := new(action.Configuration)
	if err = actionConfig.Init(restClientGetter, dep.Namespace, "secret", logger.Sugar().Debugf); err != nil {
		return nil, err
	}

	statusCli := action.NewStatus(actionConfig)
	rel, err := statusCli.Run(dep.DeploymentName)
	if err != nil {
		// release 不存在：视为 running（可能正在安装中或尚未创建）
		return drivers.Running(err.Error()), nil
	}

	switch rel.Info.Status {
	case release.StatusDeployed:
		allReady, anyFailed, msg, err := CheckReleaseWorkloadsReady(ctx, restClientGetter, rel.Manifest, dep.Namespace)
		if err != nil {
			// Helm release 已 deployed，但 readiness 判定失败（内部错误），直接失败便于快速暴露问题
			return drivers.Failed(fmt.Sprintf("helm readiness check error: %v", err)), nil
		}
		if anyFailed {
			return drivers.Failed(msg), nil
		}
		if !allReady {
			return drivers.Running(msg), nil
		}
		return drivers.Success(), nil
	case release.StatusFailed:
		return drivers.Failed("helm release failed"), nil
	default:
		return drivers.Running(string(rel.Info.Status)), nil
	}
}

func (d *Driver) execChart(ctx context.Context, namespace string, p *ExecutePayload, stage *model.StageSpecV1, kind string) (*drivers.ExecuteResult, error) {
	if stage == nil || !stage.Enabled {
		return drivers.Success(), nil
	}

	dep := p.Deployment
	app := p.App
	build := p.Build

	tplCtx := tpl.RenderTemplateContext(app, build, dep.Env, dep.ClusterName, p.TplOptions)

	cfg, err := DecodeConfig(stage.Data)
	if err != nil {
		return nil, err
	}

	chartNameTpl := cfg.ChartNameTemplate
	if chartNameTpl == "" {
		chartNameTpl = "{{.app_type}}"
	}
	chartName, err := tpl.ParseTemplate(chartNameTpl, tplCtx)
	if err != nil {
		return nil, fmt.Errorf("%s: chart_name_template 解析失败: %w", kind, err)
	}

	var chartVersion string
	if strings.TrimSpace(cfg.ChartVersionTemplate) != "" {
		chartVersion, err = tpl.ParseTemplate(cfg.ChartVersionTemplate, tplCtx)
		if err != nil {
			return nil, fmt.Errorf("%s: chart_version_template 解析失败: %w", kind, err)
		}
	}

	// values：由 helm driver 运行时计算（不落库）
	valuesMap, err := ParseValuesV1(d.db, app, build, dep.Env, dep.ClusterName, cfg.Values, p.TplOptions)
	if err != nil {
		return nil, fmt.Errorf("%s: values 计算失败: %w", kind, err)
	}

	// release name
	var releaseName string
	if releaseName, err = tpl.ParseTemplate(cfg.ReleaseNameTemplate, tplCtx); err != nil {
		return nil, fmt.Errorf("%s: release_name_template 解析失败: %w", kind, err)
	}

	param := DeploymentParam{
		AppName: app.Name,
		AppType: app.AppType,
		Values:  valuesMap,

		ReleaseName: releaseName,
		Env:         dep.Env,
		Namespace:   namespace,

		Kubeconfig: dep.Cluster.Kubeconfig,

		ChartName:    chartName,
		ChartVersion: chartVersion,
		ChartRepoURL: cfg.RepoURL,
	}

	// chart repo 认证（v1：仅 basic_auth；credential_ref 支持 "id:123" 或 "123"）
	if strings.TrimSpace(cfg.CredentialRef) != "" {
		if u, p, err := d.resolveBasicAuth(cfg.CredentialRef); err == nil {
			param.ChartUsername = u
			param.ChartPassword = p
		}
	}

	if err := NewHelmDeployer(nil).Deploy(ctx, &param); err != nil {
		return drivers.Failed(err.Error()), err
	}
	return drivers.Success(), nil
}

func (d *Driver) resolveBasicAuth(ref string) (string, string, error) {
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
	if err := d.db.First(&c, id).Error; err != nil {
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
	return m["username"], m["password"], nil
}
