package batch

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
	"time"
)

type ActionHandle func(batchId int64) error

type ActionUpdate func(batch *model.Batch, operator, reason string)

type Action struct {
	To int8

	// 查询/验证, todo: 在这里单独执行还是放在transitionHandle中执行
	Handle ActionHandle

	// 更新batch数据
	Update ActionUpdate
}

func (sm *StateMachine) initActions() {
	sm.actions = map[string]Action{
		"seal": {
			To: constants.BatchStatusSealed,
			Handle: func(batchID int64) error {
				// 1. 查询批次中的所有应用
				var releaseApps []model.ReleaseApp
				if err := sm.db.Where("batch_id = ?", batchID).Find(&releaseApps).Error; err != nil {
					return fmt.Errorf("查询%s失败: %w", model.ReleaseApp.TableName, err)
				}

				// 2. 检查应用数量（空批次不允许封板）
				if len(releaseApps) == 0 {
					return fmt.Errorf("封板失败: 批次中没有应用，不允许封板")
				}

				// 3. 检查是否所有应用都有构建
				var appsWithoutBuild []int64
				for _, app := range releaseApps {
					if app.BuildID == nil {
						appsWithoutBuild = append(appsWithoutBuild, app.AppID)
					}
				}
				if len(appsWithoutBuild) > 0 {
					return fmt.Errorf("封板失败: 以下应用没有构建记录，不允许封板: %v", appsWithoutBuild)
				}

				return nil
			},
			Update: func(batch *model.Batch, operator, reason string) {
				// 记录封板时间/操作人
				now := time.Now()
				batch.SealedAt = &now
				batch.SealedAt = &batch.UpdatedAt
				batch.SealedBy = &operator
			},
		},
		"cancel": {
			To: constants.BatchStatusCancelled,
			Handle: func(batchID int64) error {
				return nil
			},
			Update: func(batch *model.Batch, operator, reason string) {
				// 记录取消时间/操作人
				now := time.Now()
				batch.CancelledAt = &now
				batch.CancelledBy = &operator
				batch.CancelReason = &reason
			},
		},
		"start_pre_deploy": {
			To: constants.BatchStatusPreWaiting,
			Handle: func(batchID int64) error {
				var batch model.Batch
				if err := sm.db.First(&batch, batchID).Error; err != nil {
					return fmt.Errorf("查询%s失败: %w", model.ReleaseApp.TableName, err)
				}

				// 1. 检查前置条件：必须已审批通过
				if batch.ApprovalStatus != constants.ApprovalStatusApproved &&
					batch.ApprovalStatus != constants.ApprovalStatusSkipped {
					return fmt.Errorf("预发布失败: 批次未审批通过，当前审批状态: %s", batch.ApprovalStatus)
				}

				// 2. 检查前置条件：必须已封板
				if batch.Status < constants.BatchStatusSealed || batch.SealedAt == nil {
					return fmt.Errorf("预发布失败: 批次未封板")
				}

				return nil
			},
			Update: func(batch *model.Batch, operator, reason string) {
				// 记录时间/触发人
				now := time.Now()
				batch.PreStartedAt = &now
				batch.PreTriggeredBy = &operator
			},
		},
		"start_prod_deploy": {
			To: constants.BatchStatusProdWaiting,
			Handle: func(batchID int64) error {
				// todo
				return nil
			},
			Update: func(batch *model.Batch, operator, reason string) {
				// 记录时间/触发人
				now := time.Now()
				batch.ProdStartedAt = &now
				batch.ProdTriggeredBy = &operator
			},
		},
		"prod_acceptance": {
			To: constants.BatchStatusCompleted,
			Handle: func(batchID int64) error {
				return nil
			},
			Update: func(batch *model.Batch, operator, reason string) {
				// 记录时间/操作人
				now := time.Now()
				batch.FinalAcceptedAt = &now
				batch.FinalAcceptedBy = &operator
			},
		},
	}
}

// ProcessStateChange 触发状态更新, 外部调用层
func (sm *StateMachine) ProcessStateChange(batchID int64, event string, operator, reason string) error {
	e, ok := sm.actions[event]
	if !ok {
		return fmt.Errorf("无效的状态转换事件: %s", event)
	}

	// handle处理
	if e.Handle != nil {
		if err := e.Handle(batchID); err != nil {
			sm.logger.Error("状态转换处理失败", zap.Error(err))
			return err
		}
	}

	// 事务更新
	return sm.UnifiedUpdate(context.TODO(), &model.Batch{ID: batchID}, e.To, TransitionSourceOutside, func(batch *model.Batch) {
		e.Update(batch, operator, reason)
	})
}
