package core

import (
	"devops-cd/internal/core/release_app"
	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// ProcessBatchEvent 处理批次事件
func (e *CoreEngine) ProcessBatchEvent(batchID int64, event string, operator, reason string) error {
	return e.batchSM.ProcessStateChange(batchID, event, operator, reason)
}

// GetBatchStatus 获取批次状态
func (e *CoreEngine) GetBatchStatus(batchID int64) (map[string]interface{}, error) {
	var batch model.Batch
	if err := e.db.First(&batch, batchID).Error; err != nil {
		return nil, fmt.Errorf("查询批次失败: %w", err)
	}

	stateName := constants.BatchStatusToString(batch.Status)

	return map[string]interface{}{
		"batch_id":           batch.ID,
		"batch_number":       batch.BatchNumber,
		"current_state":      batch.Status,
		"current_state_name": stateName,
		"available_events":   "[]",
	}, nil
}

// NewTag 处理新 Build 事件
func (e *CoreEngine) NewTag(appID int64, build *model.Build) {
	log := e.logger.Sugar().With(zap.Int64("app_id", appID), zap.Int64("build_id", build.ID))

	// 查看该app在哪个开放的batch中 status < BatchStatusFinalAccepted
	var releases []model.ReleaseApp
	if err := e.db.Debug().InnerJoins("Join release_batches ON release_apps.batch_id = release_batches.id AND release_batches.status < ?", constants.BatchStatusFinalAccepted).
		Where("release_apps.app_id = ?", appID).
		Preload("Batch").Find(&releases).Error; err != nil {
		log.Errorf("查询应用发布记录失败: %v", err)
		return
	}

	if len(releases) == 0 {
		log.Infof("应用 %v 不在任何未封板批次中，无需更新", appID)
		return
	}
	if len(releases) > 1 {
		log.Warnf("应用 %v 在多个未封板批次中: %v", appID, releases)
	}
	release := releases[0]
	log = log.With(zap.Int64("batch_id", release.BatchID), zap.Int64("release_id", release.ID))

	if release.Batch.Status < constants.BatchStatusSealed { // 1. 如果批次未封板

		if release.BuildID == nil || release.LatestBuildID == nil || *release.BuildID == *release.LatestBuildID {
			// 1.1 如果没有build_id, 或者build_id == latest_build_id
			if err := e.releaseSM.UpdateStatus(context.TODO(), release.ID, release_app.WithModelEffects(func(r *model.ReleaseApp) {
				r.BuildID = &build.ID
				r.TargetTag = &build.ImageTag
				r.LatestBuildID = &build.ID
			})); err != nil {
				log.Errorf("更新发布记录失败: %v", err)
				return
			}
			log.Infof("[Build] Batch%v App:%v 更新发布记录成功: #%v %v", release.BatchID, release.AppID, build.ID, build.ImageTag)
		} else if release.BuildID != nil && *release.BuildID != *release.LatestBuildID {
			// 1.2 build_id != latest_build_id, 说明人工修改过, 不需要修改
		}

	} else if release.Batch.Status >= constants.BatchStatusSealed { // 2. 如果批次已封板, 只更新 latest_build_id

		if err := e.releaseSM.UpdateStatus(context.TODO(), release.ID, release_app.WithModelEffects(func(r *model.ReleaseApp) {
			r.LatestBuildID = &build.ID
		})); err != nil {
			log.Errorf("更新发布记录失败: %v", err)
			return
		}
	}

}

// SwitchVersion 切换版本
func (e *CoreEngine) SwitchVersion(req *dto.SwitchVersionRequest) (*string, error) {
	return nil, e.releaseSM.SwitchVersion(req.ReleaseAppID, req.BuildID, req.Operator, req.Reason)
}

// ManualDeploy 手动部署
func (e *CoreEngine) ManualDeploy(req *dto.ManualDeployRequest) (string, error) {
	return "ok", e.releaseSM.ManualDeploy(req.ReleaseAppID, req.Action, req.Operator, req.Reason)
}
