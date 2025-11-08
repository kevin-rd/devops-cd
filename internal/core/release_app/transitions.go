package release_app

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
)

type TransitionHandler interface {
	// Handle 检查合法性, 处理强依赖操作
	Handle(release *model.ReleaseApp, from, to int8, options *transitionOptions) error

	// After 状态转换成功后, 异步操作
	After(release *model.ReleaseApp, from, to int8, options *transitionOptions)
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

func (sm *ReleaseStateMachine) registerTransitions() {
	var transitions = []StateTransition{
		// 重新预发布
		{
			From:        0, // todo
			To:          constants.ReleaseAppStatusPreWaiting,
			Handler:     TriggerManualPreDeploy{sm: sm},
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

type TriggerManualPreDeploy struct {
	sm *ReleaseStateMachine
}

func (h TriggerManualPreDeploy) Handle(release *model.ReleaseApp, from, to int8, options *transitionOptions) error {
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

	// 2. 更新release_app状态
	release.BuildID = release.LatestBuildID
	release.TargetTag = &latestBuild.ImageTag
	// 重新发布时不需要检查依赖关系?
	release.Status = constants.ReleaseAppStatusPreCanTrigger

	return nil
}

func (h TriggerManualPreDeploy) After(release *model.ReleaseApp, from, to int8, options *transitionOptions) {
}
