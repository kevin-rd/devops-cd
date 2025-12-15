package transitions

import (
	"devops-cd/internal/model"
	"gorm.io/gorm"
	"time"
)

// OnPreDeployCompletedTransition 处理预发布部署完成
type OnPreDeployCompletedTransition struct {
	db *gorm.DB
}

func (h OnPreDeployCompletedTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {
	now := time.Now()
	batch.PreFinishedAt = &now
	return nil
}
func (h OnPreDeployCompletedTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
