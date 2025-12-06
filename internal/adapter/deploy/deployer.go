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
}
