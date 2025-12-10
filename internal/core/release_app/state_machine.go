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
		log.Warnf("[ReleaseApp SM: %d-%d] 未知 ReleaseApp 状态: %v", release.BatchID, release.ID, release.Status)
		return
	}

	// 2. 执行handle处理
	nextStatus, updateFunc, err := handler.Handle(ctx, release)
	if err != nil {
		handlerName := handler.Name()
		log.Errorf("[ReleaseApp SM: %d-%d] %s 处理失败: %v", release.BatchID, release.ID, handlerName, err)

		// 使用'即时闭包'包裹原始updateFunc
		f := func(name string, err error, before func(*model.ReleaseApp)) func(*model.ReleaseApp) {
			return func(r *model.ReleaseApp) {
				if before != nil {
					before(r)
				}
				r.AppendReasonf("%s 状态处理失败: %v", name, err)
			}
		}(handlerName, err, updateFunc)

		if err = sm.UpdateStatus(ctx, release.ID, WithToFunc(func(r model.ReleaseApp) int8 { return model.ToFailed(r.Status) }), WithModelEffects(f)); err != nil {
			log.Errorf("[ReleaseApp SM: %d-%d] [db] 状态更新失败: %v", release.BatchID, release.ID, err)
		}
		return
	}

	// 3. 状态更新
	if nextStatus != nil || updateFunc != nil {
		log.Debugf("[ReleaseApp SM: %d-%d] [UpdateStatus] Handler: %T 状态更新: %v -> %v", release.BatchID, release.ID, handler, release.Status, nextStatus)
		if err = sm.UpdateStatus(ctx, release.ID, WithStatus(nextStatus), WithModelEffects(updateFunc)); err != nil {
			log.Errorf("[ReleaseApp SM: %d-%d] [db] 状态更新失败: %v", release.BatchID, release.ID, err)
		}
	}
}

// ================== 状态更新 ==================

type TransitionOption func(*transitionOptions)

type transitionOptions struct {
	operator         string // 操作人
	operationExplain string // 操作说明

	source int8                          // 源状态
	toFunc func(r model.ReleaseApp) int8 // 目标状态

	data       map[string]interface{}    // 额外数据
	sideEffect func(r *model.ReleaseApp) // 模型更新
}

// WithStatus 设置目标状态
func WithStatus(to *int8) TransitionOption {
	return func(o *transitionOptions) {
		if to != nil {
			o.toFunc = func(r model.ReleaseApp) int8 {
				return *to
			}
		}
	}
}
func WithToFunc(f func(r model.ReleaseApp) int8) TransitionOption {
	return func(o *transitionOptions) {
		o.toFunc = f
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
func WithOperationExplain(operator string, reason string) TransitionOption {
	return func(o *transitionOptions) {
		o.operator = operator
		o.operationExplain = reason
	}
}

func WithOperatorAndReason(operator, reason string) TransitionOption {
	return func(o *transitionOptions) {
		o.operator = operator
		o.operationExplain = reason
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

func (sm *ReleaseStateMachine) UpdateStatus(ctx context.Context, releaseAppID int64, opts ...TransitionOption) error {
	option := newTransitionOptions(opts...)

	var old int8
	var to int8
	var afterHandler func()

	err := sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 重新加载最新状态
		var rel model.ReleaseApp
		if err := tx.First(&rel, releaseAppID).Error; err != nil {
			return err
		}
		old = rel.Status

		log := sm.logger.Sugar().With(zap.Int64("batch_id", rel.BatchID), zap.Int64("release_id", rel.ID))

		// 3. 应用业务字段更新
		if option.sideEffect != nil {
			option.sideEffect(&rel) // Changed from option.sideEffect(rel) to option.sideEffect(&rel)
		}

		if option.toFunc != nil {
			to = option.toFunc(rel)

			// 2. 检查是否允许
			h, ok := sm.canTransition(old, to, option.source)
			if !ok {
				return fmt.Errorf("当前状态 %v 不允许转换到 %v", old, to)
			}

			if h != nil {
				// 4. 处理强依赖操作, 失败自动回滚
				if err := h.Handle(&rel, old, option); err != nil {
					return err
				}

				afterHandler = func() {
					h.After(&rel, old, option) // Changed from h.After(rel, old, option) to h.After(&rel, old, option)
				}
			}

			rel.Status = to
		}

		// 5. 条件更新
		if old == to {
			// 状态不变，只更新业务字段
			result := tx.Model(rel).Where("id = ?", rel.ID).Save(rel)
			if result.Error != nil {
				return fmt.Errorf("状态更新失败: %v", result.Error)
			}
			if result.RowsAffected == 0 {
				log.Warn("no update or record not found")
			}
			log.Infof("[ReleaseApp SM: %v-%v] 状态更新成功", rel.BatchID, rel.ID)
		} else {
			// 状态变更，乐观锁
			result := tx.Model(rel).Where("id = ? AND status = ?", rel.ID, old).Save(rel)
			if result.Error != nil {
				return fmt.Errorf("状态更新失败: %v", result.Error)
			}
			if result.RowsAffected == 0 {
				log.Warn("no update or record not found")
			}
			log.Infof("[ReleaseApp SM: %v-%v] 状态变更成功: %v -> %v", rel.BatchID, rel.ID, old, to)
		}
		return nil
	})

	if err == nil && afterHandler != nil {
		afterHandler()
	}
	return err
}
