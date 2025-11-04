package batch

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"time"
)

type TransitionHandler interface {
	// Handle 处理状态转换, 事务执行
	Handle(batch *model.Batch, from, to int8) error

	// After 状态转换后执行, 异步
	After(batch *model.Batch, from, to int8)
}

type TransitionHandleFunc func(batch *model.Batch, from, to int8) error

func (h TransitionHandleFunc) Handle(batch *model.Batch, from, to int8) error {
	return h(batch, from, to)
}

func (h TransitionHandleFunc) After(batch *model.Batch, from, to int8) {

}

type StateTransition struct {
	From    int8
	To      int8
	Event   string
	Handler TransitionHandler

	AllowSource int8 // 使用位运算
}

// 状态流转来源: 内部/外部
const (
	TransitionSourceInside  int8 = 1 << 0
	TransitionSourceOutside int8 = 1 << 1
)

// TransitionContext 转换上下文
type TransitionContext struct {
	to        int8
	Operator  string
	Timestamp time.Time
	Data      map[string]interface{}
	source    int8
}

func (sm *StateMachine) registerTransitions() {
	var transitions = []StateTransition{
		// 草稿 -> 已封板
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusSealed,
			Handler:     TransitionHandleFunc(sm.handleSeal),
			AllowSource: TransitionSourceOutside,
		},
		// 草稿 -> 取消
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusCancelled,
			Handler:     TransitionHandleFunc(sm.handleCancel),
			AllowSource: TransitionSourceOutside,
		},
		// 已封板 -> 触发预发布（需要检查审批状态）
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusPreWaiting,
			Handler:     TransitionHandleFunc(sm.handleStartPreDeploy),
			AllowSource: TransitionSourceOutside,
		},
		// 已封板 -> 取消
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusCancelled,
			Handler:     TransitionHandleFunc(sm.handleCancel),
			AllowSource: TransitionSourceOutside,
		},
		// 预发布部署中 -> 预发布已部署 todo: internal
		{
			From:    constants.BatchStatusPreDeploying,
			To:      constants.BatchStatusPreDeployed,
			Handler: TransitionHandleFunc(sm.handlePreDeployCompleted),
		},
		// 预发布已部署 -> 触发生产
		{
			From:        constants.BatchStatusPreDeployed,
			To:          constants.BatchStatusProdWaiting,
			Handler:     TransitionHandleFunc(sm.handleStartProdDeploy),
			AllowSource: TransitionSourceOutside,
		},
		// 正式发布部署中 -> 正式发布已部署 // todo: internal
		{
			From:    constants.BatchStatusProdDeploying,
			To:      constants.BatchStatusProdDeployed,
			Handler: TransitionHandleFunc(sm.handleProdDeployCompleted),
		},
		// 正式发布已部署 -> 已完成
		{
			From:        constants.BatchStatusProdDeployed,
			To:          constants.BatchStatusCompleted,
			Handler:     TransitionHandleFunc(sm.handleComplete),
			AllowSource: TransitionSourceOutside,
		},
		// 正式发布已部署 -> 预发布已部署 (验收失败回滚)
		{
			From:    constants.BatchStatusProdDeployed,
			To:      constants.BatchStatusPreDeployed,
			Handler: TransitionHandleFunc(sm.handleProdVerifyFailed),
		},
	}

	for _, t := range transitions {
		if sm.transitions[t.From] == nil {
			sm.transitions[t.From] = make(map[int8]StateTransition)
		}
		sm.transitions[t.From][t.To] = t
	}
}

// canTransition 检查是否可以进行状态转换
func (sm *StateMachine) canTransition(from, to int8, source int8) (TransitionHandler, bool) {
	if transitions, ok := sm.transitions[from]; ok {
		if transition, ok := transitions[to]; ok && transition.AllowSource&source != 0 {
			return transition.Handler, true
		}
	}

	// 内部默认允许
	if source == TransitionSourceInside {
		return nil, true
	}

	return nil, false
}

// ================== 状态转换处理函数 ==================

// handleSeal 处理封板
func (sm *StateMachine) handleSeal(batch *model.Batch, from, to int8) error {

	// 1. 记录部署前版本（从 applications.deployed_tag 获取）
	if err := sm.db.Exec(`
		UPDATE release_apps ra
		JOIN applications a ON ra.app_id = a.id
		SET ra.previous_deployed_tag = COALESCE(a.deployed_tag, '')
		WHERE ra.batch_id = ? AND ra.is_locked = false
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录部署前版本失败: %w", err)
	}

	// 2. 记录目标版本（从 build.image_tag 获取并固定）todo
	if err := sm.db.Exec(`
		UPDATE release_apps ra
		JOIN builds b ON ra.build_id = b.id
		SET ra.target_tag = b.image_tag
		WHERE ra.batch_id = ? AND ra.build_id IS NOT NULL
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录目标版本失败: %w", err)
	}

	// 3. 锁定所有应用记录（防止封板后修改）
	if err := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Update("is_locked", true).Error; err != nil {
		return fmt.Errorf("锁定应用记录失败: %w", err)
	}

	return nil
}

// handleCancel 处理取消批次
func (sm *StateMachine) handleCancel(batch *model.Batch, from, to int8) error {
	// TODO: 实现取消逻辑
	// 1. 停止所有进行中的任务
	// 2. 清理资源
	// 3. 发送通知
	return nil
}

// handleStartPreDeploy 处理开始预发布部署
func (sm *StateMachine) handleStartPreDeploy(batch *model.Batch, from, to int8) error {
	return nil
}

// handlePreDeployCompleted 处理预发布部署完成
func (sm *StateMachine) handlePreDeployCompleted(batch *model.Batch, from, to int8) error {
	// TODO: 实现预发布部署完成逻辑
	// 1. 验证部署结果
	// 2. 记录完成时间
	// 3. 发送验收通知
	now := time.Now()
	batch.PreFinishedAt = &now
	return nil
}

// handlePreDeployFailed 处理预发布部署失败
func (sm *StateMachine) handlePreDeployFailed(batch *model.Batch, from, to int8) error {
	// TODO: 实现预发布部署失败逻辑
	// 1. 记录失败原因
	// 2. 回滚部署
	// 3. 发送告警通知
	return nil
}

// handleStartProdDeploy 处理开始正式发布部署
func (sm *StateMachine) handleStartProdDeploy(batch *model.Batch, from, to int8) error {
	// TODO: 实现开始正式发布部署逻辑
	// 1. 创建生产部署任务
	// 2. 触发生产部署流程
	// 3. 更新批次状态
	return nil
}

// handleProdDeployCompleted 处理正式发布部署完成
func (sm *StateMachine) handleProdDeployCompleted(batch *model.Batch, from, to int8) error {
	// TODO: 实现正式发布部署完成逻辑
	// 1. 验证部署结果
	// 2. 记录完成时间
	// 同步更新 applications.deployed_tag 为 target_tag（部署成功后的版本）
	if err := sm.db.Exec(`
		UPDATE applications a
		JOIN release_apps ra ON a.id = ra.app_id
		SET a.deployed_tag = ra.target_tag
		WHERE ra.batch_id = ? AND ra.target_tag IS NOT NULL
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("更新应用部署版本失败: %w", err)
	}
	// 3. 发送验收通知
	now := time.Now()
	batch.ProdFinishedAt = &now
	return nil
}

// handleProdDeployFailed 处理正式发布部署失败
func (sm *StateMachine) handleProdDeployFailed(batch *model.Batch, from, to int8) error {
	// TODO: 实现正式发布部署失败逻辑
	// 1. 记录失败原因
	// 2. 触发回滚流程
	// 3. 发送告警通知
	return nil
}

// handlePreVerifyFailed 处理预发布验收失败
func (sm *StateMachine) handlePreVerifyFailed(batch *model.Batch, from, to int8) error {
	// TODO: 实现预发布验收失败逻辑
	// 1. 记录验收失败原因
	// 2. 回滚到封板状态
	// 3. 发送通知
	return nil
}

// handleProdVerifyFailed 处理生产验收失败
func (sm *StateMachine) handleProdVerifyFailed(batch *model.Batch, from, to int8) error {
	// TODO: 实现生产验收失败逻辑
	// 1. 记录验收失败原因
	// 2. 回滚到预发布状态
	// 3. 发送告警通知
	return nil
}

// handleFinalAccept 处理最终验收通过
func (sm *StateMachine) handleComplete(batch *model.Batch, from, to int8) error {
	// 记录最终验收时间和验收人
	now := time.Now()
	batch.FinalAcceptedAt = &now
	// batch.FinalAcceptedBy = &ctx.Operator // todo
	return nil
}
