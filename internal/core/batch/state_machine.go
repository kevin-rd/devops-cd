package batch

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type StateMachine struct {
	db          *gorm.DB
	logger      *zap.Logger
	handlers    map[int8]Handler
	transitions map[int8][]StateTransition
}

func NewBatchStateMachine(db *gorm.DB, logger *zap.Logger) *StateMachine {
	sm := &StateMachine{
		db:          db,
		logger:      logger,
		handlers:    make(map[int8]Handler),
		transitions: make(map[int8][]StateTransition),
	}
	sm.registerHandlers()
	sm.initTransitions()
	return sm
}

// Process 入口
func (sm *StateMachine) Process(ctx context.Context, batch *model.Batch) {
	log := sm.logger.Sugar().With(zap.Int64("batch_id", batch.ID))

	// 1. 获取当前handler
	handler, exists := sm.handlers[batch.Status]
	if !exists {
		log.Warnf("Batch:%v(%s) -> 未知状态: %v", batch.ID, batch.BatchNumber, batch.Status)
		return
	}

	// 2. 执行handler
	nextStatus, updateFunc, err := handler.Handle(ctx, batch)
	if err != nil {
		log.Errorf("Batch:%v(%s) -> 状态处理失败 [%d]: %v", batch.ID, batch.BatchNumber, batch.Status, err)
		return
	}

	// 3. 状态更新
	if nextStatus != 0 && nextStatus != batch.Status {
		if err := sm.UnifiedUpdate(ctx, batch, nextStatus, StateSourceInside, updateFunc); err != nil {
			sm.logger.Error(fmt.Sprintf("Batch:%v(%s) -> 状态流转失败 %d→%d: %v", batch.ID, batch.BatchNumber, batch.Status, nextStatus, err))
			return
		}
	}
}

func (sm *StateMachine) UnifiedUpdate(ctx context.Context, batch *model.Batch, to, source int8, updateFunc func(*model.Batch)) error {
	return sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		log := sm.logger.Sugar().With(zap.Int64("batch_id", batch.ID))

		// 1. 重新加载最新状态
		if err := tx.First(batch, batch.ID).Error; err != nil {
			return err
		}
		old := batch.Status

		// 2. 检查是否允许
		handler, ok := sm.canTransition(batch.Status, to, source)
		if !ok {
			return fmt.Errorf("当前状态 %s 不允许转换到 %s", constants.BatchStatusToString(batch.Status), constants.BatchStatusToString(to))
		}
		// 处理强依赖操作, 失败自动回滚
		if handler != nil {
			if err := handler.Handle(batch, batch.Status, to); err != nil {
				return err
			}
		}

		// 3. 应用业务字段更新（无论是否变更状态）
		if updateFunc != nil {
			updateFunc(batch)
		}

		// 4. 条件更新
		var result *gorm.DB
		if old == to {
			// 状态未改变
			result = tx.Model(batch).Where("id = ?", batch.ID).Save(batch)
		} else {
			// 状态变更，乐观锁
			batch.Status = to
			result = tx.Model(batch).Where("id = ? AND status = ?", batch.ID, old).Save(batch)
		}
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("update failed: status conflict")
		}

		log.Infof("Batch:%v 状态变更成功: %v -> %v", batch.ID, old, to)
		//batch.NextCheckAt = time.Now().Add(30 * time.Second) // 根据层级调整
		return nil
	})
}

// 外部方法

// events are event -> 目标status
var events = map[string]int8{
	"seal":              constants.BatchStatusSealed,      // 封板确认
	"cancel":            constants.BatchStatusCancelled,   // 批次取消
	"start_pre_deploy":  constants.BatchStatusPreWaiting,  // 开始预发布部署
	"start_prod_deploy": constants.BatchStatusProdWaiting, // 开始生产部署
	"prod_acceptance":   constants.BatchStatusCompleted,   // 生产验收完成
}

// ProcessStateChange 处理批次状态变更, 对外接口
func (sm *StateMachine) ProcessStateChange(batchID int64, event string, operator string) error {
	to, ok := events[event]
	if !ok {
		return fmt.Errorf("无效的状态转换事件: %s", event)
	}

	return sm.UnifiedUpdate(context.TODO(), &model.Batch{ID: batchID}, to, StateSourceOutside, nil)
}
