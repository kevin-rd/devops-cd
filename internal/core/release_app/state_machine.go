package release_app

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ReleaseStateMachine struct {
	db       *gorm.DB
	logger   *zap.Logger
	handlers map[int8]Handler
	resolver *Resolver

	transitions map[int8]map[int8]StateTransition
}

func NewReleaseStateMachine(db *gorm.DB, logger *zap.Logger, resolver *Resolver) *ReleaseStateMachine {
	sm := &ReleaseStateMachine{
		db:          db,
		logger:      logger,
		handlers:    make(map[int8]Handler),
		transitions: make(map[int8]map[int8]StateTransition),
		resolver:    resolver,
	}
	sm.registerHandlers()
	sm.registerTransitions()
	return sm
}

// Process 入口
func (sm *ReleaseStateMachine) Process(ctx context.Context, release *model.ReleaseApp) {
	log := sm.logger.Sugar().With(zap.Int64("batch_id", release.BatchID), zap.Int64("release_id", release.ID))

	// 1. 查询当前的handler
	handler, ok := sm.handlers[release.Status]
	if !ok {
		if release.Status <= constants.ReleaseAppStatusTagged {
			return
		}
		sm.logger.Warn("未知 ReleaseApp 状态", zap.Int64("release_id", release.ID), zap.Int8("status", release.Status))
		return
	}

	// 2. 执行handle处理
	nextStatus, updateFunc, err := handler.Handle(ctx, release)
	if err != nil {
		log.Errorf("[ReleaseApp SM] Batch:%v ReleaseApp:%v 处理失败: %v", release.BatchID, release.ID, err)
		return
	}

	// 3. 状态更新
	if nextStatus != nil || updateFunc != nil {
		log.Debugf("[ReleaseApp SM] [UpdateStatus] Batch:%v ReleaseApp:%v Haandler: %T 状态更新: %v -> %v", release.BatchID, release.ID, handler, release.Status, nextStatus)
		if err = sm.UpdateStatus(ctx, release, WithStatus(nextStatus), WithModelEffects(updateFunc)); err != nil {
			log.Errorf("[ReleaseApp SM] [db] 状态更新失败: %v", err)
		}
	}
}

// ================== 状态更新 ==================

type TransitionOption func(*transitionOptions)

type transitionOptions struct {
	operator   string
	reason     string
	getTarget  func(r model.ReleaseApp) int8
	source     int8
	data       map[string]interface{}
	sideEffect func(r *model.ReleaseApp)
}

// WithStatus 设置目标状态
func WithStatus(to *int8) TransitionOption {
	return func(o *transitionOptions) {
		if to != nil {
			o.getTarget = func(r model.ReleaseApp) int8 {
				return *to
			}
		}
	}
}
func WithTarget(f func(r model.ReleaseApp) int8) TransitionOption {
	return func(o *transitionOptions) {
		o.getTarget = f
	}
}

func WithSource(source int8) TransitionOption {
	return func(o *transitionOptions) {
		o.source = source
	}
}
func WithModelEffects(sideEffects func(*model.ReleaseApp)) TransitionOption {
	if sideEffects == nil {
		return nil
	}
	return func(o *transitionOptions) { o.sideEffect = sideEffects }
}
func WithOperator(operator string) TransitionOption {
	return func(o *transitionOptions) { o.operator = operator }
}
func WithReason(reason string) TransitionOption {
	return func(o *transitionOptions) { o.reason = reason }
}
func WithOperatorAndReason(operator, reason string) TransitionOption {
	return func(o *transitionOptions) {
		o.operator = operator
		o.reason = reason
	}
}
func WithData(key string, value interface{}) TransitionOption {
	return func(o *transitionOptions) {
		if o.data == nil {
			o.data = make(map[string]interface{})
		}
		o.data[key] = value
	}
}
func newTransitionOptions(opts ...TransitionOption) *transitionOptions {
	option := &transitionOptions{source: 1}
	for _, opt := range opts {
		if opt != nil {
			opt(option)
		}
	}
	return option
}

func (sm *ReleaseStateMachine) UpdateStatus(ctx context.Context, release *model.ReleaseApp, opts ...TransitionOption) error {
	log := sm.logger.Sugar().With(zap.Int64("batch_id", release.BatchID), zap.Int64("release_id", release.ID))

	option := newTransitionOptions(opts...)

	var old int8
	var to int8
	var afterHandler func()

	err := sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 重新加载最新状态
		if err := tx.First(release, release.ID).Error; err != nil {
			return err
		}
		old = release.Status

		// 3. 应用业务字段更新
		if option.sideEffect != nil {
			option.sideEffect(release)
		}

		if option.getTarget != nil {
			to = option.getTarget(*release)

			// 2. 检查是否允许
			h, ok := sm.canTransition(old, to, option.source)
			if !ok {
				return fmt.Errorf("当前状态 %v 不允许转换到 %v", old, to)
			}

			if h != nil {
				// 4. 处理强依赖操作, 失败自动回滚
				if err := h.Handle(release, old, option); err != nil {
					return err
				}

				afterHandler = func() {
					h.After(release, old, option)
				}
			}

			release.Status = to
		}

		// 5. 条件更新
		var result = tx.Model(release).Where("id = ?", release.ID).Save(release)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("update failed: status conflict or record not found")
		}

		log.Infof("Batch:%v ReleaseApp:%v 状态变更成功: %v -> %v", release.BatchID, release.ID, old, to)
		return nil
	})

	if err == nil && afterHandler != nil {
		afterHandler()
	}
	return err
}
