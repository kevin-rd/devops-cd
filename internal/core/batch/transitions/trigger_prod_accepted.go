package transitions

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"gorm.io/gorm"
)

type TriggerProdAccepted struct {
	db *gorm.DB
}

func (h *TriggerProdAccepted) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {

	// 检查当前状态是否是 ProdDeployed
	if batch.Status != constants.BatchStatusProdDeployed {
		return fmt.Errorf("batch status is not prod-deployed")
	}

	// todo: 权限检查

	return nil
}

func (h *TriggerProdAccepted) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
