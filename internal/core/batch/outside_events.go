package batch

import (
	"context"
	"devops-cd/internal/core/batch/transitions"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
)

type Event struct {
	To int8
}

var events = map[string]Event{
	constants.BatchActionSeal:   {To: constants.BatchStatusSealed},
	constants.BatchActionCancel: {To: constants.BatchStatusCancelled},
	// pre
	constants.BatchActionStartPre:  {To: constants.BatchStatusPreWaiting},
	constants.BatchActionAcceptPre: {To: constants.BatchStatusPreAccepted},
	// prod
	constants.BatchActionStartProd:  {To: constants.BatchStatusProdWaiting},
	constants.BatchActionAcceptProd: {To: constants.BatchStatusProdAccepted},
	// final accept
	constants.BatchActionComplete: {To: constants.BatchStatusCompleted},
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
	if err := sm.ChangeStatus(context.TODO(), &model.Batch{BaseModel: model.BaseModel{ID: batchID}}, e.To, transitions.SourceOutside,
		transitions.WithOperator(operator),
		transitions.WithReason(reason),
	); err != nil {
		sm.logger.Sugar().Errorf("处理批次操作：%v失败： %v", event, err)
		return err
	}

	sm.logger.Sugar().Infof("处理批次操作：%v by %v 成功", event, operator)
	return nil
}
