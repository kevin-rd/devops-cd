package batch

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"time"
)

type TransitionHandler interface {
	// Handle 检查合法性, 处理强依赖操作
	Handle(batch *model.Batch, from, to int8, options *transitionOptions) error

	// After 状态转换成功后, 异步操作
	After(batch *model.Batch, from, to int8, options *transitionOptions)
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

type TransitionOption func(*transitionOptions)

type transitionOptions struct {
	operator string
	reason   string
	// data       map[string]interface{}
	sideEffect func(b *model.Batch)
}

func WithModelEffects(sideEffects func(b *model.Batch)) TransitionOption {
	return func(o *transitionOptions) { o.sideEffect = sideEffects }
}
func WithOperator(operator string) TransitionOption {
	return func(o *transitionOptions) { o.operator = operator }
}
func WithReason(reason string) TransitionOption {
	return func(o *transitionOptions) { o.reason = reason }
}

func (sm *StateMachine) registerTransitions() {
	var transitions = []StateTransition{
		// 草稿 -> 已封板
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusSealed,
			Handler:     TriggerSealTransition{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 草稿 -> 取消
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusCancelled,
			Handler:     TriggerCancelTransition{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 已封板 -> 触发预发布（需要检查审批状态）
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusPreWaiting,
			Handler:     TriggerPreDeployTransition{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 已封板 -> 取消
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusCancelled,
			Handler:     TriggerCancelTransition{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 预发布部署中 -> 预发布已部署 todo: internal
		{
			From:        constants.BatchStatusPreDeploying,
			To:          constants.BatchStatusPreDeployed,
			Handler:     OnPreDeployCompletedTransition{sm: sm},
			AllowSource: TransitionSourceInside,
		},
		// 预发布已部署 -> 触发生产
		{
			From:        constants.BatchStatusPreDeployed,
			To:          constants.BatchStatusProdWaiting,
			Handler:     TriggerProdDeployTransition{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		{
			From:        constants.BatchStatusPreDeployed,
			To:          constants.BatchStatusCancelled,
			Handler:     TriggerCancelTransition{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 正式发布部署中 -> 正式发布已部署 // todo: internal
		{
			From:    constants.BatchStatusProdDeploying,
			To:      constants.BatchStatusProdDeployed,
			Handler: OnProdDeployCompletedTransition{sm: sm},
		},
		// 正式发布已部署 -> 已完成
		{
			From:        constants.BatchStatusProdDeployed,
			To:          constants.BatchStatusCompleted,
			Handler:     FinalAcceptTransition{sm: sm},
			AllowSource: TransitionSourceOutside,
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

// TriggerSealTransition 处理封板
type TriggerSealTransition struct {
	sm *StateMachine
}

func (h TriggerSealTransition) Handle(batch *model.Batch, from, to int8, options *transitionOptions) error {
	// 1. 查询批次中的所有应用
	var releaseApps []model.ReleaseApp
	if err := h.sm.db.Where("batch_id = ?", batch.ID).Find(&releaseApps).Error; err != nil {
		return fmt.Errorf("查询%s失败: %w", model.ReleaseApp{}.TableName(), err)
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

	// 1. 记录部署前版本（从 applications.deployed_tag 获取）
	if err := h.sm.db.Exec(`
		UPDATE release_apps ra
		JOIN applications a ON ra.app_id = a.id
		SET ra.previous_deployed_tag = COALESCE(a.deployed_tag, '')
		WHERE ra.batch_id = ? AND ra.is_locked = false
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录部署前版本失败: %w", err)
	}

	// 2. 记录目标版本（从 build.image_tag 获取并固定）todo
	if err := h.sm.db.Exec(`
		UPDATE release_apps ra
		JOIN builds b ON ra.build_id = b.id
		SET ra.target_tag = b.image_tag
		WHERE ra.batch_id = ? AND ra.build_id IS NOT NULL
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录目标版本失败: %w", err)
	}

	// 3. 锁定所有应用记录（防止封板后修改）
	if err := h.sm.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batch.ID).
		Update("status", constants.ReleaseAppStatusTagged).
		Update("is_locked", true).Error; err != nil {
		return fmt.Errorf("锁定应用记录失败: %w", err)
	}

	// 4. 计算并固化 skip_pre_env 标记
	type ReleaseAppEnvInfo struct {
		ReleaseAppID int64
		AppID        int64
		SkipPreEnv   bool
	}

	var releaseAppEnvInfos []ReleaseAppEnvInfo
	err := h.sm.db.Raw(`
		SELECT 
			ra.id as release_app_id,
			ra.app_id,
			NOT EXISTS(
				SELECT 1 FROM app_env_configs 
				WHERE app_id = ra.app_id 
				AND env = 'pre' 
				AND status = 1
				AND deleted_at IS NULL
			) as skip_pre_env
		FROM release_apps ra
		WHERE ra.batch_id = ?
	`, batch.ID).Scan(&releaseAppEnvInfos).Error

	if err != nil {
		return fmt.Errorf("查询应用环境配置失败: %w", err)
	}

	// 批量更新 skip_pre_env
	for _, info := range releaseAppEnvInfos {
		if err := h.sm.db.Model(&model.ReleaseApp{}).
			Where("id = ?", info.ReleaseAppID).
			Update("skip_pre_env", info.SkipPreEnv).Error; err != nil {
			return fmt.Errorf("更新 skip_pre_env 失败: %w", err)
		}
	}

	h.sm.logger.Info(fmt.Sprintf("Batch:%d 封板完成,计算了 %d 个应用的环境配置", batch.ID, len(releaseAppEnvInfos)))

	// 记录时间/操作人
	now := time.Now()
	batch.SealedAt = &now
	batch.SealedBy = &options.operator

	return nil
}

func (h TriggerSealTransition) After(batch *model.Batch, from, to int8, options *transitionOptions) {
	// todo: send notification
}

// TriggerCancelTransition 处理取消批次
type TriggerCancelTransition struct {
	sm *StateMachine
}

func (h TriggerCancelTransition) Handle(batch *model.Batch, from, to int8, options *transitionOptions) error {
	now := time.Now()
	batch.CancelledAt = &now
	batch.CancelledBy = &options.operator
	batch.CancelReason = &options.reason
	return nil
}

func (h TriggerCancelTransition) After(batch *model.Batch, from, to int8, options *transitionOptions) {
	// todo: send notification
}

// TriggerPreDeployTransition 处理开始预发布部署
type TriggerPreDeployTransition struct {
	sm *StateMachine
}

func (h TriggerPreDeployTransition) Handle(batch *model.Batch, from, to int8, options *transitionOptions) error {
	// 1. 检查前置条件：必须已审批通过
	if batch.ApprovalStatus != constants.ApprovalStatusApproved &&
		batch.ApprovalStatus != constants.ApprovalStatusSkipped {
		return fmt.Errorf("预发布失败: 批次未审批通过，当前审批状态: %s", batch.ApprovalStatus)
	}

	// 2. 检查前置条件：必须已封板
	if batch.Status < constants.BatchStatusSealed || batch.SealedAt == nil {
		return fmt.Errorf("预发布失败: 批次未封板")
	}

	// 3. todo: 所有的app都已经tagged

	return nil
}
func (h TriggerPreDeployTransition) After(batch *model.Batch, from, to int8, options *transitionOptions) {
	now := time.Now()
	batch.PreStartedAt = &now
	batch.PreTriggeredBy = &options.operator
}

// handlePreDeployCompleted 处理预发布部署完成
type OnPreDeployCompletedTransition struct {
	sm *StateMachine
}

func (h OnPreDeployCompletedTransition) Handle(batch *model.Batch, from, to int8, options *transitionOptions) error {
	now := time.Now()
	batch.PreFinishedAt = &now
	return nil
}
func (h OnPreDeployCompletedTransition) After(batch *model.Batch, from, to int8, options *transitionOptions) {
	// todo: send notification
}

// handlePreDeployFailed 处理预发布部署失败
func (sm *StateMachine) handlePreDeployFailed(batch *model.Batch, from, to int8) error {
	// TODO: 实现预发布部署失败逻辑
	// 1. 记录失败原因
	// 2. 回滚部署
	// 3. 发送告警通知
	return nil
}

// TriggerProdDeployTransition 处理正式发布部署完成
type TriggerProdDeployTransition struct {
	sm *StateMachine
}

func (h TriggerProdDeployTransition) Handle(batch *model.Batch, from, to int8, options *transitionOptions) error {
	// TODO: 实现开始正式发布部署逻辑
	// 1. 创建生产部署任务
	// 2. 触发生产部署流程
	// 3. 更新批次状态
	return nil
}
func (h TriggerProdDeployTransition) After(batch *model.Batch, from, to int8, options *transitionOptions) {
	now := time.Now()
	batch.ProdStartedAt = &now
	batch.ProdTriggeredBy = &options.operator
}

// handleProdDeployCompleted 处理正式发布部署完成
type OnProdDeployCompletedTransition struct {
	sm *StateMachine
}

func (h OnProdDeployCompletedTransition) Handle(batch *model.Batch, from, to int8, options *transitionOptions) error {
	// 同步更新 applications.deployed_tag 为 target_tag（部署成功后的版本）
	if err := h.sm.db.Exec(`
		UPDATE applications a
		JOIN release_apps ra ON a.id = ra.app_id
		SET a.deployed_tag = ra.target_tag
		WHERE ra.batch_id = ? AND ra.target_tag IS NOT NULL
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("更新应用部署版本失败: %w", err)
	}

	now := time.Now()
	batch.ProdFinishedAt = &now
	return nil
}
func (h OnProdDeployCompletedTransition) After(batch *model.Batch, from, to int8, options *transitionOptions) {
	// todo: send notification
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

// FinalAcceptTransition 处理最终验收
type FinalAcceptTransition struct {
	sm *StateMachine
}

func (h FinalAcceptTransition) Handle(batch *model.Batch, from, to int8, options *transitionOptions) error {
	// 记录时间/操作人
	now := time.Now()
	batch.FinalAcceptedAt = &now
	batch.FinalAcceptedBy = &options.operator
	return nil
}

func (h FinalAcceptTransition) After(batch *model.Batch, from, to int8, options *transitionOptions) {
	// todo: send notification
}
