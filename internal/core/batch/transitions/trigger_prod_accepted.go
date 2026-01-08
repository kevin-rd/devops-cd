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

	// 必须全部 ProdDeployed
	var totalCount int64
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Count(&totalCount).Error; err != nil {
		return err
	}
	if totalCount == 0 {
		return fmt.Errorf("批次无发布记录，无法进行生产验收")
	}

	var prodDeployedCount int64
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Where("status = ?", constants.ReleaseAppStatusProdDeployed).
		Count(&prodDeployedCount).Error; err != nil {
		return err
	}
	if prodDeployedCount != totalCount {
		return fmt.Errorf("有未完成生产部署的应用，无法验收")
	}

	// 批量更新：ProdDeployed -> ProdAccepted
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Where("status = ?", constants.ReleaseAppStatusProdDeployed).
		Update("status", constants.ReleaseAppStatusProdAccepted).Error; err != nil {
		return err
	}

	return nil
}

func (h *TriggerProdAccepted) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
