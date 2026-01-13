package service

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/logger"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

// GetReleaseApp 获取单个发布应用详情
func (s *BatchService) GetReleaseApp(releaseAppID int64) (*dto.ReleaseAppResponse, error) {
	log := logger.Log.With(zap.Int64("release_app_id", releaseAppID)).Sugar()

	// 1. 获取 release_app 记录（包含关联信息）
	release, err := s.batchRepo.GetReleaseAppByID(releaseAppID)
	if err != nil {
		return nil, fmt.Errorf("发布应用不存在: %w", err)
	}

	releaseResp := &dto.ReleaseAppResponse{
		ID:      release.ID,
		BatchID: release.BatchID,
		AppID:   release.AppID,
		BuildID: release.BuildID,

		// 版本信息
		PreviousDeployedTag: release.PreviousDeployedTag,
		TargetTag:           release.TargetTag,
		LatestBuildID:       release.LatestBuildID,

		// 发布信息
		ReleaseNotes: release.ReleaseNotes,
		IsLocked:     release.IsLocked,
		SkipPreEnv:   release.SkipPreEnv,
		Reasons:      release.GetRecentReason(10),
		Status:       release.Status,

		CreatedAt: release.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: release.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 1.1 填充应用信息
	if release.Application != nil {
		releaseResp.AppName = release.Application.Name
		releaseResp.AppType = release.Application.AppType
		//releaseResp.AppProject = release.Application.Namespace
		releaseResp.TeamID = release.Application.TeamID
		releaseResp.DeployedTag = release.Application.DeployedTag // 当前部署的标签
		releaseResp.DefaultDependsOn = release.Application.DefaultDependsOn

		// 1.2 填充仓库信息
		releaseResp.RepoID = release.Application.RepoID
		if release.Application.Repository != nil {
			releaseResp.RepoName = release.Application.Repository.Name
			releaseResp.RepoFullName = release.Application.Repository.Namespace + "/" + release.Application.Repository.Name
		}

		// 1.3 填充团队信息
		if release.Application.Team != nil {
			releaseResp.TeamName = &release.Application.Team.Name
		}
	}

	// 2. 加载构建信息
	if release.BuildID != nil {
		build, err := s.batchRepo.GetBuildByID(*release.BuildID)
		if err != nil {
			log.Errorf("查询构建失败: %v", err)
		}

		if build != nil {
			releaseResp.BuildNumber = &build.BuildNumber
			releaseResp.BuildStatus = &build.BuildStatus
			buildTime := build.BuildCreated.Format("2006-01-02T15:04:05Z07:00")
			releaseResp.BuildTime = &buildTime
			releaseResp.ImageURL = &build.ImageURL
			releaseResp.CommitSHA = &build.CommitSHA
			releaseResp.CommitMessage = &build.CommitMessage
			releaseResp.CommitBranch = &build.CommitBranch

			// 2.1 加载最新的构建记录
			if builds, err := s.batchRepo.GetBuildsSinceTime(release.AppID, build.CreatedAt, 10); err != nil {
				log.Errorf("查询最近构建失败: %v", err)
			} else {
				releaseResp.RecentBuilds = s.toBuildSummaries(builds)
			}
		} else {
			log.Warnf("build_id不存在")
			// 2.1 加载最近的构建记录
			//if builds, err := s.batchRepo.GetRecentBuilds(release.AppID, 10); err != nil {
			//	log.Errorf("查询最近构建失败: %v", err)
			//} else {
			//	releaseResp.RecentBuilds = s.toBuildSummaries(builds)
			//}
		}
	}

	// 3. 加载 deployments（deployment 维度信息）
	var deployments []model.Deployment
	if err := s.db.
		Where("release_id = ? AND superseded_by IS NULL", release.ID).
		Order("env ASC, cluster ASC, id DESC").
		Find(&deployments).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("查询 deployments 失败: %v", err)
	} else if len(deployments) > 0 {
		resp := make([]dto.DeploymentResponse, 0, len(deployments))
		for _, dep := range deployments {
			var startedAt *string
			if dep.StartedAt != nil {
				s := dep.StartedAt.Format(time.RFC3339)
				startedAt = &s
			}
			var finishedAt *string
			if dep.FinishedAt != nil {
				s := dep.FinishedAt.Format(time.RFC3339)
				finishedAt = &s
			}

			resp = append(resp, dto.DeploymentResponse{
				ID: dep.ID,

				BatchID:   dep.BatchID,
				ReleaseID: dep.ReleaseID,
				AppID:     dep.AppID,

				Env:            dep.Env,
				ClusterName:    dep.ClusterName,
				Namespace:      dep.Namespace,
				DeploymentName: dep.DeploymentName,
				DriverType:     dep.DriverType,
				Status:         dep.Status,
				RetryCount:     dep.RetryCount,
				MaxRetryCount:  dep.MaxRetryCount,
				ErrorMessage:   dep.ErrorMessage,

				StartedAt:  startedAt,
				FinishedAt: finishedAt,
				CreatedAt:  dep.CreatedAt.Format(time.RFC3339),
				UpdatedAt:  dep.UpdatedAt.Format(time.RFC3339),
			})
		}
		releaseResp.Deployments = resp
	}

	return releaseResp, nil
}

// UpdateBuilds 更新批次应用的构建版本
func (s *BatchService) UpdateBuilds(req *dto.UpdateBuildsRequest) error {
	// 1. 获取批次
	batch, err := s.batchRepo.GetByID(req.BatchID)
	if err != nil {
		return fmt.Errorf("批次不存在: %w", err)
	}

	// 2. 检查批次状态（只能修改草稿状态的批次）
	if batch.Status >= constants.BatchStatusSealed {
		return fmt.Errorf("只能修改草稿状态的批次")
	}

	// 3. 验证所有的app_id和build_id
	if len(req.BuildChanges) == 0 {
		return fmt.Errorf("没有需要更新的构建")
	}

	// 4. 使用事务更新
	err = s.db.Transaction(func(tx *gorm.DB) error {
		for appID, buildID := range req.BuildChanges {
			// 4.1 检查release_app是否存在
			var releaseApp model.ReleaseApp
			if err := tx.Where("batch_id = ? AND app_id = ?", req.BatchID, appID).First(&releaseApp).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return fmt.Errorf("应用 %d 不在批次中", appID)
				}
				return fmt.Errorf("查询应用失败: %w", err)
			}

			// 4.2 检查build是否存在且属于该应用
			var build model.Build
			if err := tx.Where("id = ? AND app_id = ?", buildID, appID).First(&build).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return fmt.Errorf("构建 %d 不存在或不属于应用 %d", buildID, appID)
				}
				return fmt.Errorf("查询构建失败: %w", err)
			}

			// 4.3 检查构建状态（只能选择成功的构建）
			if build.BuildStatus != "success" {
				return fmt.Errorf("构建 %d 状态为 %s，只能选择成功的构建", buildID, build.BuildStatus)
			}

			// 4.4 更新release_app的build_id和target_tag
			if err := tx.Model(&releaseApp).Updates(map[string]interface{}{
				"build_id":   buildID,
				"target_tag": build.ImageTag,
			}).Error; err != nil {
				return fmt.Errorf("更新应用构建失败: %w", err)
			}

			logger.Info("更新应用构建成功",
				zap.Int64("batch_id", req.BatchID),
				zap.Int64("app_id", appID),
				zap.Int64("build_id", buildID),
				zap.String("image_tag", build.ImageTag),
				zap.String("operator", req.Operator))
		}

		return nil
	})

	if err != nil {
		return err
	}

	logger.Info("批量更新应用构建成功",
		zap.Int64("batch_id", req.BatchID),
		zap.Int("update_count", len(req.BuildChanges)),
		zap.String("operator", req.Operator))

	return nil
}

// RetryDeployment 手动重试 deployment（仅 failed 可重试）
func (s *BatchService) RetryDeployment(deploymentID int64, operator string, reason string) error {
	if deploymentID <= 0 {
		return fmt.Errorf("deployment_id 无效")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var dep model.Deployment
		if err := tx.Where("id = ?", deploymentID).First(&dep).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("deployment 不存在")
			}
			return err
		}

		if dep.SupersededBy != nil {
			return fmt.Errorf("deployment 已被替代，禁止重试")
		}
		if dep.Status != constants.DeploymentStatusFailed {
			return fmt.Errorf("仅 failed 状态允许重试，当前状态=%s", dep.Status)
		}

		updates := map[string]any{
			"status":        constants.DeploymentStatusPending,
			"retry_count":   dep.RetryCount + 1,
			"error_message": nil,
			"started_at":    nil,
			"finished_at":   nil,
		}

		if err := tx.Model(&model.Deployment{}).Where("id = ? AND status = ?", dep.ID, constants.DeploymentStatusFailed).Updates(updates).Error; err != nil {
			return err
		}

		logger.Info("手动重试 deployment",
			zap.Int64("deployment_id", dep.ID),
			zap.Int64("batch_id", dep.BatchID),
			zap.Int64("release_id", dep.ReleaseID),
			zap.Int64("app_id", dep.AppID),
			zap.String("operator", operator),
			zap.String("reason", reason))

		return nil
	})
}
