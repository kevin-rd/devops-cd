package transitions

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"gorm.io/gorm"
	"time"
)

// FinalAcceptTransition 处理最终验收, PM验收
type FinalAcceptTransition struct {
	db *gorm.DB
}

func (h FinalAcceptTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {
	// 最终验收前：必须全部 ProdAccepted
	var totalCount int64
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Count(&totalCount).Error; err != nil {
		return err
	}
	if totalCount == 0 {
		return fmt.Errorf("批次无发布记录，无法最终验收")
	}
	var prodAcceptedCount int64
	if err := h.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND status = ?", batch.ID, constants.ReleaseAppStatusProdAccepted).
		Count(&prodAcceptedCount).Error; err != nil {
		return err
	}
	if prodAcceptedCount != totalCount {
		return fmt.Errorf("有未验收的生产环境，无法最终验收")
	}

	// 记录时间/操作人
	now := time.Now()
	batch.FinalAcceptedAt = &now
	batch.FinalAcceptedBy = &options.operator
	return nil
}

func (h FinalAcceptTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
