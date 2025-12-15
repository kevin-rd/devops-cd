package transitions

import (
	"devops-cd/internal/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

// OnProdDeployCompletedTransition Prod部署完成时调用
type OnProdDeployCompletedTransition struct {
	db *gorm.DB
}

func (h OnProdDeployCompletedTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {
	// 同步更新 applications.deployed_tag 为 target_tag（部署成功后的版本）
	if err := h.db.Exec(`
		UPDATE applications a
		JOIN release_apps ra ON a.id = ra.app_id
		SET a.deployed_tag = ra.target_tag
		WHERE ra.batch_id = ? AND ra.target_tag IS NOT NULL
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("更新应用部署版本失败: %w", err)
	}

	now := time.Now()
	batch.ProdFinishedAt = &now
	return nil
}
func (h OnProdDeployCompletedTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
