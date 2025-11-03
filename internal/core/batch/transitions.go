package batch

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"time"
)

type TransitionHandler interface {
	Handle(batch *model.Batch, from, to int8) error
}

type TransitionHandleFunc func(batch *model.Batch, from, to int8) error

func (h TransitionHandleFunc) Handle(batch *model.Batch, from, to int8) error {
	return h(batch, from, to)
}

type StateTransition struct {
	From        int8
	To          int8
	Event       string
	Description string
	Handler     TransitionHandler

	AllowSource int8 // 使用位运算
}

const (
	StateSourceInside  int8 = 1 << 0
	StateSourceOutside int8 = 1 << 1
)

// TransitionContext 转换上下文
type TransitionContext struct {
	Operator  string
	Reason    string
	Timestamp time.Time
	Data      map[string]interface{}
}

func (sm *StateMachine) initTransitions() {
	transitions := []StateTransition{
		// 草稿 -> 已封板
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusSealed,
			Event:       "seal",
			Description: "封板确认",
			Handler:     TransitionHandleFunc(sm.handleSeal),
			AllowSource: StateSourceOutside,
		},
		// 草稿 -> 取消
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusCancelled,
			Event:       "cancel",
			Description: "取消批次",
			Handler:     TransitionHandleFunc(sm.handleCancel),
			AllowSource: StateSourceOutside,
		},
		// 已封板 -> 触发预发布（需要检查审批状态）
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusPreWaiting,
			Event:       "start_pre_deploy",
			Description: "开始预发布部署",
			Handler:     TransitionHandleFunc(sm.handleStartPreDeploy),
			AllowSource: StateSourceOutside,
		},
		// 已封板 -> 取消
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusCancelled,
			Event:       "cancel",
			Description: "取消批次",
			Handler:     TransitionHandleFunc(sm.handleCancel),
			AllowSource: StateSourceOutside,
		},
		// 预发布部署中 -> 预发布已部署 todo: internal
		{
			From:        constants.BatchStatusPreDeploying,
			To:          constants.BatchStatusPreDeployed,
			Event:       "pre_deploy_completed",
			Description: "预发布部署完成",
			Handler:     TransitionHandleFunc(sm.handlePreDeployCompleted),
		},
		// 预发布已部署 -> 触发生产
		{
			From:        constants.BatchStatusPreDeployed,
			To:          constants.BatchStatusProdWaiting,
			Event:       "start_prod_deploy",
			Description: "开始正式发布部署",
			Handler:     TransitionHandleFunc(sm.handleStartProdDeploy),
			AllowSource: StateSourceOutside,
		},
		// 正式发布部署中 -> 正式发布已部署 // todo: internal
		{
			From:        constants.BatchStatusProdDeploying,
			To:          constants.BatchStatusProdDeployed,
			Event:       "prod_deploy_completed",
			Description: "正式发布部署完成",
			Handler:     TransitionHandleFunc(sm.handleProdDeployCompleted),
		},
		// 正式发布已部署 -> 已完成
		{
			From:        constants.BatchStatusProdDeployed,
			To:          constants.BatchStatusCompleted,
			Event:       "complete",
			Description: "发布完成",
			Handler:     TransitionHandleFunc(sm.handleComplete),
			AllowSource: StateSourceOutside,
		},
		// 正式发布已部署 -> 预发布已部署 (验收失败回滚)
		{
			From:        constants.BatchStatusProdDeployed,
			To:          constants.BatchStatusPreDeployed,
			Event:       "prod_verify_failed",
			Description: "生产验收失败",
			Handler:     TransitionHandleFunc(sm.handleProdVerifyFailed),
		},
	}

	// 构建转换映射
	for _, t := range transitions {
		sm.transitions[t.From] = append(sm.transitions[t.From], t)
	}
}

// canTransition 检查是否可以进行状态转换
func (sm *StateMachine) canTransition(from, to int8, source int8) (TransitionHandler, bool) {

	if transitions, ok := sm.transitions[from]; ok {
		for _, t := range transitions {
			if t.To == to && t.AllowSource&source != 0 {
				// 找到并允许执行
				return t.Handler, true
			}
		}
	}

	// 内部默认允许
	if source == StateSourceInside {
		return nil, true
	}

	return nil, false
}

func (sm *StateMachine) publishEvent(batch *model.Batch, old, new int8) {
	// 可集成 webhook / kafka / prometheus
	sm.logger.Info(fmt.Sprintf("Batch:%v(%s) 状态变更: %d → %d", batch.ID, batch.BatchNumber, old, new))
}

// ================== 状态转换处理函数 ==================

// handleSeal 处理封板（原 handleConfirmTag）
func (sm *StateMachine) handleSeal(batch *model.Batch, from, to int8) error {
	// 1. 查询批次中的所有应用
	var releaseApps []model.ReleaseApp
	if err := sm.db.Where("batch_id = ?", batch.ID).Find(&releaseApps).Error; err != nil {
		return fmt.Errorf("查询批次应用失败: %w", err)
	}

	// 2. 检查应用数量（空批次不允许封板）
	if len(releaseApps) == 0 {
		return fmt.Errorf("封板失败: 批次中没有应用，不允许封板")
	}

	// 3. 检查是否所有应用都有构建
	appsWithoutBuild := []int64{}
	for _, app := range releaseApps {
		if app.BuildID == nil {
			appsWithoutBuild = append(appsWithoutBuild, app.AppID)
		}
	}

	if len(appsWithoutBuild) > 0 {
		return fmt.Errorf("封板失败: 以下应用没有构建记录，不允许封板: %v", appsWithoutBuild)
	}

	// 4. 记录版本历史信息
	// 4.1 记录部署前版本（从 applications.deployed_tag 获取）
	if err := sm.db.Exec(`
		UPDATE release_apps ra
		JOIN applications a ON ra.app_id = a.id
		SET ra.previous_deployed_tag = COALESCE(a.deployed_tag, '')
		WHERE ra.batch_id = ? AND ra.is_locked = false
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录部署前版本失败: %w", err)
	}

	// 4.2 记录目标版本（从 build.image_tag 获取并固定）todo
	if err := sm.db.Exec(`
		UPDATE release_apps ra
		JOIN builds b ON ra.build_id = b.id
		SET ra.target_tag = b.image_tag
		WHERE ra.batch_id = ? AND ra.build_id IS NOT NULL
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录目标版本失败: %w", err)
	}

	// 5. 锁定所有应用记录（防止封板后修改）
	if err := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Update("is_locked", true).Error; err != nil {
		return fmt.Errorf("锁定应用记录失败: %w", err)
	}

	// 6. 记录封板时间
	now := time.Now()
	batch.TaggedAt = &now

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
	// 1. 检查前置条件：必须已审批通过
	if batch.ApprovalStatus != constants.ApprovalStatusApproved &&
		batch.ApprovalStatus != constants.ApprovalStatusSkipped {
		return fmt.Errorf("预发布失败: 批次未审批通过，当前审批状态: %s", batch.ApprovalStatus)
	}

	// 2. 检查前置条件：必须已封板
	if batch.Status < constants.BatchStatusSealed || batch.TaggedAt == nil {
		return fmt.Errorf("预发布失败: 批次未封板")
	}

	// 3. 记录预发布开始时间
	now := time.Now()
	batch.PreDeployStartedAt = &now

	return nil
}

// handlePreDeployCompleted 处理预发布部署完成
func (sm *StateMachine) handlePreDeployCompleted(batch *model.Batch, from, to int8) error {
	// TODO: 实现预发布部署完成逻辑
	// 1. 验证部署结果
	// 2. 记录完成时间
	// 3. 发送验收通知
	now := time.Now()
	batch.PreDeployFinishedAt = &now
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
	now := time.Now()
	batch.ProdDeployStartedAt = &now
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
	batch.ProdDeployFinishedAt = &now
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
