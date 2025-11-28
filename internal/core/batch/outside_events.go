package batch

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
	constants.BatchActionSeal:      {To: constants.BatchStatusSealed},
	constants.BatchActionCancel:    {To: constants.BatchStatusCancelled},
	constants.BatchActionStartPre:  {To: constants.BatchStatusPreWaiting},
	constants.BatchActionStartProd: {To: constants.BatchStatusProdWaiting},
	constants.BatchActionComplete:  {To: constants.BatchStatusCompleted},
}

func (sm *StateMachine) initActions() {

}

// ProcessStateChange 触发状态更新, 外部调用层
func (sm *StateMachine) ProcessStateChange(batchID int64, event string, operator, reason string) error {
	e, ok := events[event]
	if !ok {
		return fmt.Errorf("无效的状态转换事件: %s", event)
	}

	// 事务更新
	return sm.ChangeStatus(context.TODO(), &model.Batch{BaseModel: model.BaseModel{ID: batchID}}, e.To, TransitionSourceOutside,
		WithOperator(operator),
		WithReason(reason),
	)
}
