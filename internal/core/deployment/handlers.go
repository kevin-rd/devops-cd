package deployment

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
	"time"
)

type Handler interface {
	Handle(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error)
}

type HandlerFunc func(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error)

func (h HandlerFunc) Handle(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	return h(ctx, dep)
}

func (sm *StateMachine) registerHandlers() {
	sm.handlers[constants.DeploymentStatusPending] = HandlerFunc(sm.HandlePending)
	sm.handlers[constants.DeploymentStatusRunning] = HandlerFunc(sm.HandleRunning)
	sm.handlers[constants.DeploymentStatusWaitingDependencies] = HandlerFunc(sm.HandleWaiting)
}

// handlers

// HandlePending handle Pending -> Running
func (sm *StateMachine) HandlePending(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	if sm.resolver != nil {
		result, err := sm.resolver.CheckDeployment(ctx, dep)
		if err != nil {
			return "", nil, err
		}

		if result.HasFailed() {
			reason := result.Summary()
			return constants.DeploymentStatusFailed, func(d *model.Deployment) {
				d.TaskID = dep.TaskID
				setErrorMessage(d, reason)
			}, nil
		}

		if result.HasPending() {
			reason := result.Summary()
			return constants.DeploymentStatusWaitingDependencies, func(d *model.Deployment) {
				d.TaskID = dep.TaskID
				setErrorMessage(d, reason)
			}, nil
		}
	}

	// 1. 触发部署（幂等）
	if err := sm.deployer.Deploy(ctx, dep); err != nil {
		return constants.DeploymentStatusFailed, func(deployment *model.Deployment) {
			setErrorMessage(deployment, err.Error())
			deployment.RetryCount++
		}, nil
	}

	// 2. 变更为 Running
	return constants.DeploymentStatusRunning, func(d *model.Deployment) {
		setErrorMessage(d, "")
	}, nil
}

// HandleRunning handle Running → Success / Failed
func (sm *StateMachine) HandleRunning(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	log := sm.logger.With(zap.Int64("deployment_id", int64(dep.ID)))

	status, err := sm.deployer.CheckStatus(ctx, dep)
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
		log.Debug(fmt.Sprintf("[Deployment SM] Batch:%v ReleaseApp:%v 部署进行中", dep.BatchID, dep.ReleaseID), zap.String("external_status", status))
		return "", nil, nil // 继续等待
	}
}

// HandleWaiting 等待依赖满足 -> Pending
func (sm *StateMachine) HandleWaiting(ctx context.Context, dep *model.Deployment) (string, func(*model.Deployment), error) {
	if sm.resolver == nil {
		return constants.DeploymentStatusPending, nil, nil
	}

	result, err := sm.resolver.CheckDeployment(ctx, dep)
	if err != nil {
		return "", nil, err
	}

	if result.HasFailed() {
		reason := result.Summary()
		return constants.DeploymentStatusFailed, func(d *model.Deployment) {
			setErrorMessage(d, reason)
		}, nil
	}

	if result.HasPending() {
		reason := result.Summary()
		return "", func(d *model.Deployment) {
			setErrorMessage(d, reason)
		}, nil
	}

	return constants.DeploymentStatusPending, func(d *model.Deployment) {
		setErrorMessage(d, "")
	}, nil
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
