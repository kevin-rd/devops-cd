package deployment

import (
	"context"
	"devops-cd/internal/adapter/deploy"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
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

	param := deploy.DeploymentParam{
		AppName: deployment.Application.Name,
		AppType: deployment.Application.AppType,
		Values:  deployment.Values,

		ReleaseName: deployment.DeploymentName,
		Env:         deployment.Env,
		Namespace:   deployment.Namespace,

		Kubeconfig: deployment.Cluster.Kubeconfig,
	}

	return sm.deployer.Deploy(ctx, &param)
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
