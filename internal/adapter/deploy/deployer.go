package deploy

import (
	"context"
	"devops-cd/internal/model"
)

// Deployer 适配器接口
type Deployer interface {

	// Deploy deploy application
	Deploy(ctx context.Context, dep *model.Deployment) error

	// CheckStatus check deployment status
	// return running/success/failed
	CheckStatus(ctx context.Context, dep *model.Deployment) (string, error)
}
