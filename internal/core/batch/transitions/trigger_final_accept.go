package transitions

import (
	"devops-cd/internal/model"
	"gorm.io/gorm"
	"time"
)

// FinalAcceptTransition 处理最终验收, PM验收
type FinalAcceptTransition struct {
	db *gorm.DB
}

func (h FinalAcceptTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {
	// 记录时间/操作人
	now := time.Now()
	batch.FinalAcceptedAt = &now
	batch.FinalAcceptedBy = &options.operator
	return nil
}

func (h FinalAcceptTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
