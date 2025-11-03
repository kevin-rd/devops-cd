package deployment

import (
	"context"
	"devops-cd/internal/adapter/deploy"
	"devops-cd/internal/model"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type StateMachine struct {
	db       *gorm.DB
	logger   *zap.Logger
	deployer deploy.Deployer
	handlers map[string]Handler
}

func NewDeploymentStateMachine(db *gorm.DB, logger *zap.Logger, deployer deploy.Deployer) *StateMachine {
	sm := &StateMachine{db: db, logger: logger, deployer: deployer, handlers: make(map[string]Handler)}
	sm.registerHandlers()
	return sm
}

func (sm *StateMachine) Process(ctx context.Context, dep *model.Deployment) {
	handler, ok := sm.handlers[dep.Status]
	if !ok {
		sm.logger.Warn("未知 Deployment 状态", zap.Int64("id", dep.ID), zap.String("status", dep.Status))
		return
	}

	nextStatus, updateFunc, err := handler.Handle(ctx, dep)
	if err != nil {
		sm.logger.Error("处理失败", zap.Error(err))
		return
	}

	if nextStatus != "" && nextStatus != dep.Status {
		if err := sm.UnifiedUpdate(ctx, dep, nextStatus, updateFunc); err != nil {
			sm.logger.Error("更新失败", zap.Error(err))
		}
	} else if updateFunc != nil {
		// 状态不变但有字段更新
		if err := sm.UnifiedUpdate(ctx, dep, dep.Status, updateFunc); err != nil {
			sm.logger.Error("字段更新失败", zap.Error(err))
		}
	}
}

func (sm *StateMachine) UnifiedUpdate(ctx context.Context, dep *model.Deployment, to string, updateFunc func(*model.Deployment)) error {
	return sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(dep, dep.ID).Error; err != nil {
			return err
		}

		old := dep.Status
		if updateFunc != nil {
			updateFunc(dep)
		}

		if old != to {
			dep.Status = to
			// dep.NextCheckAt = time.Now().Add(10 * time.Second)
		}

		var result *gorm.DB
		if old == to {
			result = tx.Model(dep).Where("id = ?", dep.ID).Save(dep)
		} else {
			result = tx.Model(dep).Where("id = ? AND status = ?", dep.ID, old).Save(dep)
		}

		if result.Error != nil || result.RowsAffected == 0 {
			return fmt.Errorf("update failed")
		}

		sm.logger.Info(fmt.Sprintf("[Deployment SM] Batch:%v ReleaseApp:%v Deployment:%v 状态变更成功: %v -> %v", dep.BatchID, dep.ReleaseID, dep.ID, old, to),
			zap.Int64("batch_id", dep.BatchID), zap.Int64("release_id", dep.ReleaseID), zap.Int64("deployment_id", dep.ID))

		return nil
	})
}
