package release_app

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"devops-cd/pkg/utils"
	"fmt"
)

type Event struct {
	To int8
}

var events = map[string]Event{
	"manual_trigger_pre":  {To: constants.ReleaseAppStatusPreCanTrigger},
	"manual_trigger_prod": {To: constants.ReleaseAppStatusProdCanTrigger},
}

// ManualDeploy 手动触发部署
func (sm *ReleaseStateMachine) ManualDeploy(releaseAppID int64, action, operator, reason string) error {
	e, ok := events[action]
	if !ok {
		return fmt.Errorf("无效的状态转换动作: %s", action)
	}

	return sm.UpdateStatus(context.TODO(), &model.ReleaseApp{ID: releaseAppID},
		WithStatus(utils.CopyInt8(e.To)),
		WithSource(TransitionSourceOutside),
		WithOperator(operator),
		WithReason(reason),
	)
}

// SwitchVersion 切换版本
func (sm *ReleaseStateMachine) SwitchVersion(releaseAppID, buildID int64, operator, reason string) error {
	return nil
}

// TriggerPre 手动触发Pre部署
func (sm *ReleaseStateMachine) TriggerPre(releaseAppID, buildID int64, operator, reason string) error {
	// 事务更新
	return sm.UpdateStatus(context.TODO(), &model.ReleaseApp{ID: releaseAppID},
		WithStatus(utils.CopyInt8(constants.ReleaseAppStatusPreCanTrigger)),
		WithSource(TransitionSourceOutside),
		WithOperator(operator),
		WithReason(reason),
		WithData("build_id", buildID),
	)
}

// TriggerProd 手动触发Prod部署
func (sm *ReleaseStateMachine) TriggerProd(releaseAppID, buildID int64, operator, reason string) error {
	// 事务更新
	return sm.UpdateStatus(context.TODO(), &model.ReleaseApp{ID: releaseAppID},
		WithStatus(utils.CopyInt8(constants.ReleaseAppStatusPreCanTrigger)),
		WithSource(TransitionSourceOutside),
		WithOperator(operator),
		WithReason(reason),
		WithData("build_id", buildID),
	)
}
