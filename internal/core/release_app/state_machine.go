package release_app

import (
	"context"
	"devops-cd/internal/core/dependency"
	"devops-cd/internal/model"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ReleaseStateMachine struct {
	db       *gorm.DB
	logger   *zap.Logger
	handlers map[int8]Handler
	resolver *dependency.Resolver
}

func NewReleaseStateMachine(db *gorm.DB, logger *zap.Logger, resolver *dependency.Resolver) *ReleaseStateMachine {
	sm := &ReleaseStateMachine{db: db, logger: logger, handlers: make(map[int8]Handler), resolver: resolver}
	sm.registerHandlers()
	return sm
}

// Process 入口
func (sm *ReleaseStateMachine) Process(ctx context.Context, release *model.ReleaseApp) {
	// 1. 查询当前的handler
	handler, ok := sm.handlers[release.Status]
	if !ok {
		sm.logger.Warn("未知 ReleaseApp 状态", zap.Int64("release_id", release.ID), zap.Int8("status", release.Status))
		return
	}

	// 2. 执行handle处理
	nextStatus, updateFunc, err := handler.Handle(ctx, release)
	if err != nil {
		sm.logger.Error(fmt.Sprintf("[ReleaseApp SM] Batch:%v ReleaseApp:%v 处理失败: %v", release.BatchID, release.ID, err),
			zap.Int64("batch_id", release.BatchID), zap.Int64("release_id", release.ID))
		return
	}

	// 3. 状态更新
	if nextStatus != 0 {
		if err = sm.UnifiedUpdate(ctx, release, nextStatus, updateFunc); err != nil {
			sm.logger.Error(fmt.Sprintf("[ReleaseApp SM] [db] 状态更新失败: %v", err))
		}
	}
}

func (sm *ReleaseStateMachine) UnifiedUpdate(ctx context.Context, release *model.ReleaseApp, to int8, updateFunc func(*model.ReleaseApp)) error {
	return sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		log := sm.logger.Sugar().With(zap.Int64("batch_id", release.BatchID), zap.Int64("release_id", release.ID))

		// 1. 重新加载最新状态
		if err := tx.First(release, release.ID).Error; err != nil {
			return err
		}

		old := release.Status

		// 2. 应用业务字段更新（无论是否变更状态）
		if updateFunc != nil {
			updateFunc(release)
		}

		// 3. 只有状态需要变更时，才更新 status + next_check_at
		if old != to {
			release.Status = to
			//release.NextCheckAt = time.Now().Add(15 * time.Second)
		}

		// 4. 条件更新：如果状态没变，用 WHERE id=?；如果变了，用 WHERE status=old
		var result *gorm.DB
		if old == to {
			// 状态不变，只更新业务字段
			result = tx.Model(release).Where("id = ?", release.ID).Save(release)
		} else {
			// 状态变更，乐观锁
			result = tx.Model(release).Where("id = ? AND status = ?", release.ID, old).Save(release)
		}

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("update failed: status conflict or record not found")
		}

		log.Infof("Batch:%v ReleaseApp:%v 状态变更成功: %v -> %v", release.BatchID, release.ID, old, to)
		return nil
	})
}
