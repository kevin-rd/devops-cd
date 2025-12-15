package transitions

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

// TriggerSealTransition 处理封板
type TriggerSealTransition struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (h TriggerSealTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {
	// 1. 查询批次中的所有应用
	var releaseApps []model.ReleaseApp
	if err := h.db.Where("batch_id = ?", batch.ID).Find(&releaseApps).Error; err != nil {
		return fmt.Errorf("查询%s失败: %w", model.ReleaseApp{}.TableName(), err)
	}

	// 2. 检查应用数量（空批次不允许封板）
	if len(releaseApps) == 0 {
		return fmt.Errorf("封板失败: 批次中没有应用，不允许封板")
	}

	// 3. 检查是否所有应用都有构建
	var appsWithoutBuild []int64
	for _, app := range releaseApps {
		if app.BuildID == nil {
			appsWithoutBuild = append(appsWithoutBuild, app.AppID)
		}
	}
	if len(appsWithoutBuild) > 0 {
		return fmt.Errorf("封板失败: 以下应用没有构建记录，不允许封板: %v", appsWithoutBuild)
	}

	// 1. 记录部署前版本（从 applications.deployed_tag 获取）
	if err := h.db.Exec(`
		UPDATE release_apps ra
		JOIN applications a ON ra.app_id = a.id
		SET ra.previous_deployed_tag = COALESCE(a.deployed_tag, '')
		WHERE ra.batch_id = ? AND ra.is_locked = false
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录部署前版本失败: %w", err)
	}

	// 2. 记录目标版本（从 build.image_tag 获取并固定）todo
	if err := h.db.Exec(`
		UPDATE release_apps ra
		JOIN builds b ON ra.build_id = b.id
		SET ra.target_tag = b.image_tag
		WHERE ra.batch_id = ? AND ra.build_id IS NOT NULL
	`, batch.ID).Error; err != nil {
		return fmt.Errorf("记录目标版本失败: %w", err)
	}

	// 3. 锁定所有应用记录（防止封板后修改）
	if err := h.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batch.ID).
		Update("status", constants.ReleaseAppStatusTagged).
		Update("is_locked", true).Error; err != nil {
		return fmt.Errorf("锁定应用记录失败: %w", err)
	}

	// 4. 计算并固化 skip_pre_env 标记
	type ReleaseAppEnvInfo struct {
		ReleaseAppID int64
		AppID        int64
		SkipPreEnv   bool
	}

	var releaseAppEnvInfos []ReleaseAppEnvInfo
	err := h.db.Raw(`
		SELECT 
			ra.id as release_app_id,
			ra.app_id,
			NOT EXISTS(
				SELECT 1 FROM app_env_configs 
				WHERE app_id = ra.app_id 
				AND env = 'pre' 
				AND status = 1
				AND deleted_at IS NULL
			) as skip_pre_env
		FROM release_apps ra
		WHERE ra.batch_id = ?
	`, batch.ID).Scan(&releaseAppEnvInfos).Error

	if err != nil {
		return fmt.Errorf("查询应用环境配置失败: %w", err)
	}

	// 批量更新 skip_pre_env
	for _, info := range releaseAppEnvInfos {
		if err := h.db.Model(&model.ReleaseApp{}).
			Where("id = ?", info.ReleaseAppID).
			Update("skip_pre_env", info.SkipPreEnv).Error; err != nil {
			return fmt.Errorf("更新 skip_pre_env 失败: %w", err)
		}
	}

	h.logger.Info(fmt.Sprintf("Batch:%d 封板完成,计算了 %d 个应用的环境配置", batch.ID, len(releaseAppEnvInfos)))

	// 记录时间/操作人
	now := time.Now()
	batch.SealedAt = &now
	batch.SealedBy = &options.operator

	return nil
}

func (h TriggerSealTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {
	// todo: send notification
}
