package release_app

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
)

type Event struct {
	To int8
}

var events = map[string]Event{
	"new_tag":             {},
	"manual_trigger_pre":  {To: constants.ReleaseAppStatusPreCanTrigger},
	"manual_trigger_prod": {To: constants.ReleaseAppStatusProdCanTrigger},
}

// ProcessStateChange 触发状态更新, 外部调用层
func (sm *ReleaseStateMachine) ProcessStateChange(id int64, event string, operator, reason string) error {
	e, ok := events[event]
	if !ok {
		return fmt.Errorf("无效的状态转换事件: %s", event)
	}

	// 事务更新
	return sm.UpdateStatus(context.TODO(), &model.ReleaseApp{ID: id},
		WithStatus(e.To),
		WithOperator(operator),
		WithReason(reason),
	)
}
