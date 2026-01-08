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

	// 仅对需要 pre 的应用做验收：skip_pre_env=false
	var needPreCount int64
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND skip_pre_env = ?", batch.ID, false).
		Count(&needPreCount).Error; err != nil {
		return err
	}
	if needPreCount == 0 {
		return fmt.Errorf("批次中无需要预发布的应用，无法进行预发布验收")
	}

	// 必须全部 PreDeployed
	var preDeployedCount int64
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND skip_pre_env = ?", batch.ID, false).
		Where("status = ?", constants.ReleaseAppStatusPreDeployed).
		Count(&preDeployedCount).Error; err != nil {
		return err
	}
	if preDeployedCount != needPreCount {
		return fmt.Errorf("有未完成预发布部署的应用，无法验收")
	}

	// 批量更新：PreDeployed -> PreAccepted
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND skip_pre_env = ?", batch.ID, false).
		Where("status = ?", constants.ReleaseAppStatusPreDeployed).
		Update("status", constants.ReleaseAppStatusPreAccepted).Error; err != nil {
		return err
	}

	return nil
}

func (h *TriggerPreAccepted) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
