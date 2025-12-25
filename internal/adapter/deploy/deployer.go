package deploy

import (
	"context"
)

// Deployer 适配器接口, 强绑定Helm
type Deployer interface {

	// Deploy deploy application
	Deploy(ctx context.Context, param *DeploymentParam) error

	// CheckStatus check deployment status
	// return running/success/failed
	CheckStatus(ctx context.Context, param *DeploymentParam) (string, error)
}

type DeploymentParam struct {
	AppName string // Usually: Chart name is AppName or AppType
	AppType string
	Values  map[string]interface{}

	ReleaseName string
	Env         string
	Namespace   string

	Kubeconfig string

	// Chart 相关（v1：从 project_env_configs.artifacts_json 解析而来；为空时 fallback 到全局 helm.repo.*）
	ChartSourceType string // helm_repo | pipeline_artifact
	ChartName       string // 已解析后的 chart 名称
	ChartVersion    string // 可选
	ChartRepoURL    string // helm_repo
	ChartUsername   string // 可选（basic auth）
	ChartPassword   string // 可选（basic auth）

	// pipeline_artifact：chart tgz 下载地址（v1: HTTP）
	ChartArtifactURL string
}
