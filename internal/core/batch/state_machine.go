package batch

import (
	"context"
	transitions2 "devops-cd/internal/core/batch/transitions"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type StateMachine struct {
	db     *gorm.DB
	logger *zap.Logger

	// 内部Stata触发
	handlers map[int8]StateHandler

	// 状态转换
	transitions map[int8]map[int8]transitions2.StateTransition
}

func NewBatchStateMachine(db *gorm.DB, logger *zap.Logger) *StateMachine {
	sm := &StateMachine{
		db:          db,
		logger:      logger,
		handlers:    make(map[int8]StateHandler),
		transitions: make(map[int8]map[int8]transitions2.StateTransition),
	}

	sm.registerHandlers()
	sm.registerTransitions()
	return sm
}

// Process 是StateHandler
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
		if err = sm.ChangeStatus(ctx, batch, nextStatus, transitions2.SourceInside, transitions2.WithModelEffects(updateFunc)); err != nil {
			sm.logger.Error(fmt.Sprintf("Batch:%v(%s) -> 状态流转失败 %d→%d: %v", batch.ID, batch.BatchNumber, batch.Status, nextStatus, err))
			return
		}
	}
}

func (sm *StateMachine) ChangeStatus(ctx context.Context, batch *model.Batch, to, source int8, opts ...transitions2.TransitionOption) error {
	log := sm.logger.Sugar().With(zap.Int64("batch_id", batch.ID))

	option := &transitions2.TransitionOptions{}
	for _, opt := range opts {
		opt(option)
	}

	var from int8
	var afterHandler func()

	err := sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 重新加载最新状态
		if err := tx.First(batch, batch.ID).Error; err != nil {
			return err
		}
		from = batch.Status

		// 2. 检查是否允许
		h, ok := sm.canTransition(batch.Status, to, source)
		if !ok {
			return fmt.Errorf("当前状态 %s 不允许转换到 %s", constants.BatchStatusToString(batch.Status), constants.BatchStatusToString(to))
		}

		// 3. 执行业务字段更新
		if option.SideEffect != nil {
			option.SideEffect(batch)
		}

		if h != nil {
			// 4. 处理强依赖操作, 失败自动回滚
			if err := h.Handle(batch, batch.Status, to, option); err != nil {
				return err
			}

			// 后处理函数
			afterHandler = func() {
				h.After(batch, from, to, option)
			}
		}

		// 5. 乐观锁更新
		batch.Status = to
		result := tx.Model(batch).Where("id = ? AND status = ?", batch.ID, from).Save(batch)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("update failed: status conflict")
		}

		log.Infof("[Batch SM: %d] 状态变更成功: %v -> %v", batch.ID, from, to)
		// batch.NextCheckAt = time.Now().Add(30 * time.Second) // 根据层级调整
		return nil
	})

	// 事务成功后执行 after()
	if err == nil && afterHandler != nil {
		afterHandler()
	}
	return err
}

// canTransition 检查是否可以进行状态转换
func (sm *StateMachine) canTransition(from, to int8, source int8) (transitions2.TransitionHandler, bool) {
	if transitions, ok := sm.transitions[from]; ok {
		if transition, ok := transitions[to]; ok {
			if transition.AllowSource&source != 0 {
				return transition.Handler, true
			}
		}
	}

	// 内部默认允许
	if source == transitions2.SourceInside {
		return nil, true
	}

	return nil, false
}

func (sm *StateMachine) registerTransitions() {
	trans := transitions2.AllTransitions(sm.db)

	for _, t := range trans {
		if sm.transitions[t.From] == nil {
			sm.transitions[t.From] = make(map[int8]transitions2.StateTransition)
		}
		sm.transitions[t.From][t.To] = t
	}
}
