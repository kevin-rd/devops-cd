package transitions

import (
	"devops-cd/internal/model"
	"gorm.io/gorm"
	"time"
)

// TriggerCancelTransition 处理取消批次
type TriggerCancelTransition struct {
	db *gorm.DB
}

func (h TriggerCancelTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {
	now := time.Now()
	batch.CancelledAt = &now
	batch.CancelledBy = &options.operator
	batch.CancelReason = &options.reason
	return nil
}

func (h TriggerCancelTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
