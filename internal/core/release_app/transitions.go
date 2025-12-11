package release_app

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
)

type TransitionHandler interface {
	// Handle 检查合法性, 处理强依赖操作
	Handle(release *model.ReleaseApp, from int8, options *transitionOptions) error

	// After 状态转换成功后, 异步操作
	After(release *model.ReleaseApp, from int8, options *transitionOptions)
}

// StateTransition 状态转换定义
type StateTransition struct {
	From    []int8
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

func (sm *ReleaseStateMachine) registerTransitions() {
	transitions := []StateTransition{
		// 重新预发布
		{
			From: []int8{
				// PreWaiting&PreCanTrigger时不允许重新预发布, 防止并发问题, PreTriggered是否允许先待定
				constants.ReleaseAppStatusPreTriggered, constants.ReleaseAppStatusPreDeployed, constants.ReleaseAppStatusPreFailed,
				constants.ReleaseAppStatusProdTriggered, constants.ReleaseAppStatusProdDeployed, constants.ReleaseAppStatusProdFailed,
			},
			To:          constants.ReleaseAppStatusPreCanTrigger, // 跳过依赖检查
			Handler:     SwitchVersionPreDeploy{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 重新Prod发布(该App没有Pre环境)
		{
			From: []int8{
				constants.ReleaseAppStatusProdTriggered, constants.ReleaseAppStatusProdDeployed, constants.ReleaseAppStatusProdFailed,
			},
			To:          constants.ReleaseAppStatusProdCanTrigger,
			Handler:     SwitchVersionProdDeploy{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 提前Pre发布
		{
			From:        []int8{constants.ReleaseAppStatusTagged},
			To:          constants.ReleaseAppStatusPreCanTrigger,
			Handler:     ManualTriggerPreDeploy{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 手动触发Prod发布(Pre发布完成后提前触发Prod/没有Pre环境直接提交触发Prod)
		{
			From:        []int8{constants.ReleaseAppStatusTagged, constants.ReleaseAppStatusPreDeployed},
			To:          constants.ReleaseAppStatusProdCanTrigger,
			Handler:     ManualTriggerProdDeploy{sm: sm},
			AllowSource: TransitionSourceOutside,
		},
		// 生产完成
		{
			From:        []int8{constants.ReleaseAppStatusProdTriggered},
			To:          constants.ReleaseAppStatusProdDeployed,
			Handler:     OnProdDeployCompleted{sm: sm},
			AllowSource: TransitionSourceInside,
		},
	}

	for _, t := range transitions {
		fs := t.From
		for _, f := range fs {
			if sm.transitions[f] == nil {
				sm.transitions[f] = make(map[int8]StateTransition)
			}
			sm.transitions[f][t.To] = t
		}
	}
}

// canTransition 检查是否可以进行状态转换
func (sm *ReleaseStateMachine) canTransition(from, to int8, source int8) (TransitionHandler, bool) {
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

// SwitchVersionPreDeploy 切换版本
type SwitchVersionPreDeploy struct {
	sm *ReleaseStateMachine
}

func (h SwitchVersionPreDeploy) Handle(release *model.ReleaseApp, from int8, options *transitionOptions) error {
	var batch model.Batch
	if err := h.sm.db.First(&batch, release.BatchID).Error; err != nil {
		return err
	}

	var latestBuild model.Build
	if err := h.sm.db.First(&latestBuild, release.LatestBuildID).Error; err != nil {
		return fmt.Errorf("查询latestBuild记录失败: %w", err)
	}

	// 1. 预发布已完成/生产已完成时才可以手动重新执行预发布
	if batch.Status >= constants.BatchStatusPreDeployed && batch.Status < constants.BatchStatusProdWaiting ||
		batch.Status >= constants.BatchStatusProdDeployed && batch.Status < constants.BatchStatusFinalAccepted {
		// 可以进行
	} else {
		return fmt.Errorf("当前批次状态不允许手动重新预发布")
	}

	// 2. 获取build_id并更新
	targetBuild, ok := options.data["build_id"].(int64)
	if !ok || targetBuild == 0 {
		return fmt.Errorf("未指定目标构建")
	}

	var build model.Build
	if err := h.sm.db.First(&build, targetBuild).Error; err != nil {
		return fmt.Errorf("查询Build记录失败: %w", err)
	}

	release.BuildID = &build.ID
	release.TargetTag = &build.ImageTag
	// 重新发布时不需要检查依赖关系?
	release.Status = constants.ReleaseAppStatusPreCanTrigger

	return nil
}
func (h SwitchVersionPreDeploy) After(release *model.ReleaseApp, from int8, options *transitionOptions) {
}

// SwitchVersionProdDeploy 当App没有预发布环境, 直接切换Prod
type SwitchVersionProdDeploy struct {
	sm *ReleaseStateMachine
}

func (h SwitchVersionProdDeploy) Handle(release *model.ReleaseApp, from int8, options *transitionOptions) error {
	var batch model.Batch
	if err := h.sm.db.First(&batch, release.BatchID).Error; err != nil {
		return err
	}

	// 1. 检查状态
	if !release.SkipPreEnv {
		return fmt.Errorf("当前App有预发布环境, 不允许直接Prod发布")
	}

	if batch.Status >= constants.BatchStatusPreDeployed && batch.Status <= constants.BatchStatusProdWaiting ||
		batch.Status >= constants.BatchStatusProdDeployed && batch.Status <= constants.BatchStatusFinalAccepted {
		// Pre/Prod已完成时才可以手动重新Prod发布
	} else {
		return fmt.Errorf("当前Batch Status: %v 不允许手动重新Prod发布", batch.Status)
	}

	// 重新发布时不需要检查依赖关系?
	targetBuild, ok := options.data["build_id"].(int64)
	if !ok || targetBuild == 0 {
		return fmt.Errorf("未指定build_id")
	}

	var build model.Build
	if err := h.sm.db.First(&build, targetBuild).Error; err != nil {
		return fmt.Errorf("查询Build记录失败: %w", err)
	}

	release.BuildID = &build.ID
	release.TargetTag = &build.ImageTag
	release.Status = constants.ReleaseAppStatusProdCanTrigger

	return nil
}

func (h SwitchVersionProdDeploy) After(release *model.ReleaseApp, from int8, options *transitionOptions) {
}

// ManualTriggerPreDeploy 手动触发Pre发布
type ManualTriggerPreDeploy struct {
	sm *ReleaseStateMachine
}

func (h ManualTriggerPreDeploy) Handle(release *model.ReleaseApp, from int8, options *transitionOptions) error {
	var batch model.Batch
	if err := h.sm.db.First(&batch, release.BatchID).Error; err != nil {
		return err
	}

	if batch.Status < constants.BatchStatusSealed {
		return fmt.Errorf("[Pre]发布失败: 批次未封板")
	}

	if release.BuildID == nil || release.TargetTag == nil {
		return fmt.Errorf("目标版本为空, 无法进行[Pre]发布")
	}

	release.Status = constants.ReleaseAppStatusPreCanTrigger

	return nil

}
func (h ManualTriggerPreDeploy) After(release *model.ReleaseApp, from int8, options *transitionOptions) {
}

// ManualTriggerProdDeploy 手动触发Prod发布
type ManualTriggerProdDeploy struct {
	sm *ReleaseStateMachine
}

func (h ManualTriggerProdDeploy) Handle(release *model.ReleaseApp, from int8, options *transitionOptions) error {
	var batch model.Batch
	if err := h.sm.db.First(&batch, release.BatchID).Error; err != nil {
		return err
	}
	if batch.Status < constants.BatchStatusSealed {
		return fmt.Errorf("[Prod]发布失败: 批次未封板")
	}

	// 检查build
	if release.BuildID == nil || release.TargetTag == nil {
		return fmt.Errorf("目标版本为空, 无法进行[Prod]发布")
	}

	if release.Status == constants.ReleaseAppStatusTagged {
		// -> 无预发布情况
		// 检查该应用 是否无预发布环境
		if !release.SkipPreEnv {
			return fmt.Errorf("[Prod]发布失败: 请先进行预发布")
		}
	} else if release.Status == constants.ReleaseAppStatusPreDeployed {
		// -> 预发布完成情况
	} else {
		// -> 未知状态
		return fmt.Errorf("[Prod]发布失败: 批次状态异常")
	}

	release.Status = constants.ReleaseAppStatusProdCanTrigger
	return nil
}
func (h ManualTriggerProdDeploy) After(release *model.ReleaseApp, from int8, options *transitionOptions) {
}

type OnProdDeployCompleted struct {
	sm *ReleaseStateMachine
}

func (h OnProdDeployCompleted) Handle(release *model.ReleaseApp, from int8, options *transitionOptions) error {
	// todo: check all prod deployment is success

	if release.TargetTag == nil {
		return fmt.Errorf("目标版本为空, 无法更新应用部署版本")
	}

	// 1. 更新app的deployed_tag
	if err := h.sm.db.Model(&model.Application{}).
		Where("id = ?", release.AppID).Update("deployed_tag", release.TargetTag).Error; err != nil {
		return fmt.Errorf("更新应用部署版本失败: %w", err)
	}

	return nil
}

func (h OnProdDeployCompleted) After(release *model.ReleaseApp, from int8, options *transitionOptions) {
}
