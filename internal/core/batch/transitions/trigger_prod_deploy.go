package transitions

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"gorm.io/gorm"
	"time"
)

// TriggerProdDeployTransition 触发Prod部署
type TriggerProdDeployTransition struct {
	db *gorm.DB
}

func (h TriggerProdDeployTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {

	if batch.Status == constants.BatchStatusPreAccepted {
		// 当前为预发布验收完成状态: 检查所有预发布已验收
		var preAcceptedCount int64
		if err := h.db.Model(&model.ReleaseApp{}).Where("batch_id = ? AND status = ?", batch.ID, constants.BatchStatusPreAccepted).
			Count(&preAcceptedCount).Error; err != nil {
			return err
		}

		var statusInPreCount int64
		if err := h.db.Model(&model.ReleaseApp{}).Where("batch_id = ? AND skip_pre_env = ? ", batch.ID, false).
			Scopes(StatusIn(constants.BatchStatusPreWaiting)).
			Count(&statusInPreCount).Error; err != nil {
			return err
		}

		if preAcceptedCount < statusInPreCount {
			return fmt.Errorf("有未验收的预发布环境")
		}

	} else if batch.Status != constants.BatchStatusSealed {
		// 当前为封板状态: 检查是否都没有预发布环境
		// todo: 如果有app已经提前预发布, 是否允许直接prod发布
		var countInPre int64
		if err := h.db.Model(&model.ReleaseApp{}).Where("batch_id = ? AND skip_pre_env = ? ", batch.ID, false).
			Count(&countInPre).Error; err != nil {
			return err
		}
		if countInPre > 0 {
			return fmt.Errorf("有未跳过的预发布环境")
		}

		var countAll int64
		if err := h.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batch.ID).
			Count(&countAll).Error; err != nil {
			return err
		}
		if countAll == 0 {
			return fmt.Errorf("批次至少需要包含一个应用")
		}

	} else {
		return fmt.Errorf("当前状态不能触发Prod部署")
	}

	// 操作记录
	now := time.Now()
	batch.ProdStartedAt = &now
	batch.ProdTriggeredBy = &options.operator

	return nil
}
func (h TriggerProdDeployTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
}
