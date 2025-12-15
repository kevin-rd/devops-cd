package transitions

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"gorm.io/gorm"
)

type TriggerPreAccepted struct {
	db *gorm.DB
}

func (h *TriggerPreAccepted) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {

	// 检查当前状态是否是 PreDeployed
	if batch.Status != constants.BatchStatusPreDeployed {
		return fmt.Errorf("batch status is not pre-deployed")
	}

	// todo: 权限检查

	return nil
}

func (h *TriggerPreAccepted) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
