package service

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/repository"
	"devops-cd/pkg/constants"
)

// BatchService 批次服务
type BatchService struct {
	batchRepo *repository.BatchRepository
	db        *gorm.DB
}

// NewBatchService 创建批次服务
func NewBatchService(db *gorm.DB) *BatchService {
	return &BatchService{
		batchRepo: repository.NewBatchRepository(db),
		db:        db,
	}
}

// CreateBatchRequest 创建批次请求
type CreateBatchRequest struct {
	BatchNumber  string           `json:"batch_number" binding:"required"` // 批次编号/标题，用户填写
	Initiator    string           `json:"initiator" binding:"required"`    // 发起人
	ReleaseNotes *string          `json:"release_notes"`                   // 批次级发布说明（可选）
	Apps         []CreateBatchApp `json:"apps"`                            // 应用列表（允许为空，封板时校验）
}

// CreateBatchApp 批次中的应用
type CreateBatchApp struct {
	AppID        int64   `json:"app_id" binding:"required"` // 应用ID
	ReleaseNotes *string `json:"release_notes"`             // 应用级发布说明（可选）
}

// CreateBatch 创建批次
func (s *BatchService) CreateBatch(req *CreateBatchRequest) (*model.Batch, error) {
	// 1. 检查批次编号是否重复
	existBatch, err := s.batchRepo.GetByBatchNumber(req.BatchNumber)
	if err == nil && existBatch != nil {
		return nil, fmt.Errorf("批次编号 %s 已存在", req.BatchNumber)
	}

	// 2. 如果没有应用，直接创建空批次
	if len(req.Apps) == 0 {
		logger.Info("创建空批次（无应用）",
			zap.String("batch_number", req.BatchNumber),
			zap.String("initiator", req.Initiator))

		batch := &model.Batch{
			BatchNumber:    req.BatchNumber,
			Initiator:      req.Initiator,
			ReleaseNotes:   req.ReleaseNotes,
			Status:         constants.BatchStatusDraft,      // 草稿状态
			ApprovalStatus: constants.ApprovalStatusPending, // 待审批
		}
		if err := s.db.Create(batch).Error; err != nil {
			return nil, fmt.Errorf("创建批次失败: %w", err)
		}
		return batch, nil
	}

	// 3. 提取应用ID列表
	appIDs := make([]int64, len(req.Apps))
	for i, app := range req.Apps {
		appIDs[i] = app.AppID
	}

	// 4. 检查应用是否存在
	appMap, err := s.batchRepo.GetApplicationsByIDs(appIDs)
	if err != nil {
		return nil, fmt.Errorf("查询应用失败: %w", err)
	}
	if len(appMap) != len(appIDs) {
		return nil, fmt.Errorf("部分应用不存在")
	}

	// 5. 检查应用冲突（严格模式）
	conflicts, err := s.batchRepo.CheckAppConflict(appIDs, nil)
	if err != nil {
		return nil, fmt.Errorf("检查应用冲突失败: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, &AppConflictError{Conflicts: conflicts, AppMap: appMap}
	}

	// 6. 获取每个应用的最新成功构建（可能部分应用没有构建）
	buildMap, err := s.batchRepo.GetLatestSuccessBuilds(appIDs)
	if err != nil {
		return nil, fmt.Errorf("查询最新构建失败: %w", err)
	}

	// 注意：不再强制要求所有应用都有构建，允许无构建的应用加入批次
	// 无构建的应用会在封板时进行校验

	// 7. 使用事务创建批次和应用记录
	var batch *model.Batch
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 创建批次
		batch = &model.Batch{
			BatchNumber:    req.BatchNumber,
			Initiator:      req.Initiator,
			ReleaseNotes:   req.ReleaseNotes,
			Status:         constants.BatchStatusDraft,      // 草稿状态
			ApprovalStatus: constants.ApprovalStatusPending, // 待审批
		}
		if err := tx.Create(batch).Error; err != nil {
			return fmt.Errorf("创建批次失败: %w", err)
		}

		// 创建应用发布记录
		releaseApps := make([]*model.ReleaseApp, 0, len(req.Apps))
		for _, app := range req.Apps {
			build, hasBuild := buildMap[app.AppID]

			releaseApp := &model.ReleaseApp{
				BatchID:      batch.ID,
				AppID:        app.AppID,
				ReleaseNotes: app.ReleaseNotes,
				IsLocked:     false,
			}

			// 如果有构建记录，填充构建信息
			if hasBuild {
				releaseApp.BuildID = &build.ID
				releaseApp.TargetTag = &build.ImageTag
				releaseApp.LatestBuildID = &build.ID
			} else {
				// 无构建记录，留空
				releaseApp.BuildID = nil
			}

			releaseApps = append(releaseApps, releaseApp)
		}

		if err := tx.Create(&releaseApps).Error; err != nil {
			return fmt.Errorf("创建应用发布记录失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	logger.Info("批次创建成功",
		zap.String("batch_number", batch.BatchNumber),
		zap.Int64("batch_id", batch.ID),
		zap.Int("app_count", len(req.Apps)))

	return batch, nil
}

// UpdateBatchRequest 更新批次请求
type UpdateBatchRequest struct {
	BatchID      int64            `json:"batch_id" binding:"required"`
	Operator     string           `json:"operator" binding:"required"`
	BatchNumber  *string          `json:"batch_number"`
	ReleaseNotes *string          `json:"release_notes"`
	AddApps      []CreateBatchApp `json:"add_apps"` // 新增应用
	RemoveAppIDs []int64          `json:"remove_app_ids"`
}

// UpdateBatch 更新批次（通用）
func (s *BatchService) UpdateBatch(req *UpdateBatchRequest) (*model.Batch, map[string]interface{}, error) {
	// 1. 获取批次
	batch, err := s.batchRepo.GetByID(req.BatchID)
	if err != nil {
		return nil, nil, fmt.Errorf("批次不存在: %w", err)
	}

	// 2. 检查批次状态（只能修改未封板的批次）
	if batch.Status >= constants.BatchStatusSealed {
		return nil, nil, &BatchSealedError{BatchID: batch.ID, Status: batch.Status}
	}

	updatedFields := make(map[string]interface{})

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 3. 更新基本信息
		if req.BatchNumber != nil && *req.BatchNumber != batch.BatchNumber {
			// 检查新批次编号是否重复
			existBatch, err := s.batchRepo.GetByBatchNumber(*req.BatchNumber)
			if err == nil && existBatch != nil && existBatch.ID != batch.ID {
				return fmt.Errorf("批次编号 %s 已存在", *req.BatchNumber)
			}
			batch.BatchNumber = *req.BatchNumber
			updatedFields["batch_number"] = *req.BatchNumber
		}

		if req.ReleaseNotes != nil {
			batch.ReleaseNotes = req.ReleaseNotes
			updatedFields["release_notes"] = *req.ReleaseNotes
		}

		// 4. 删除应用
		if len(req.RemoveAppIDs) > 0 {
			if err := tx.Where("batch_id = ? AND app_id IN ?", batch.ID, req.RemoveAppIDs).
				Delete(&model.ReleaseApp{}).Error; err != nil {
				return fmt.Errorf("删除应用失败: %w", err)
			}
			updatedFields["remove_app_ids"] = req.RemoveAppIDs
		}

		// 5. 添加应用
		if len(req.AddApps) > 0 {
			// 提取应用ID列表
			addAppIDs := make([]int64, len(req.AddApps))
			for i, app := range req.AddApps {
				addAppIDs[i] = app.AppID
			}

			// 检查应用冲突
			conflicts, err := s.batchRepo.CheckAppConflict(addAppIDs, &batch.ID)
			if err != nil {
				return fmt.Errorf("检查应用冲突失败: %w", err)
			}
			if len(conflicts) > 0 {
				appMap, _ := s.batchRepo.GetApplicationsByIDs(addAppIDs)
				return &AppConflictError{Conflicts: conflicts, AppMap: appMap}
			}

			// 获取最新构建（可能部分应用没有构建）
			buildMap, err := s.batchRepo.GetLatestSuccessBuilds(addAppIDs)
			if err != nil {
				return fmt.Errorf("查询最新构建失败: %w", err)
			}

			// 注意：不再强制要求所有应用都有构建，允许无构建的应用加入批次
			// 无构建的应用会在封板时进行校验

			// 创建应用发布记录
			releaseApps := make([]*model.ReleaseApp, 0, len(req.AddApps))
			for _, app := range req.AddApps {
				build, hasBuild := buildMap[app.AppID]

				releaseApp := &model.ReleaseApp{
					BatchID:      batch.ID,
					AppID:        app.AppID,
					ReleaseNotes: app.ReleaseNotes,
					IsLocked:     false,
				}

				// 如果有构建记录，填充构建信息
				if hasBuild {
					releaseApp.BuildID = &build.ID
					releaseApp.TargetTag = &build.ImageTag
					releaseApp.LatestBuildID = &build.ID
				} else {
					// 无构建记录，留空
					releaseApp.BuildID = nil
				}

				releaseApps = append(releaseApps, releaseApp)
			}

			if err := tx.Create(&releaseApps).Error; err != nil {
				return fmt.Errorf("创建应用发布记录失败: %w", err)
			}
			updatedFields["add_apps"] = addAppIDs
		}

		// 6. 检查批次是否还有应用
		var appCount int64
		if err := tx.Model(&model.ReleaseApp{}).Where("batch_id = ?", batch.ID).Count(&appCount).Error; err != nil {
			return err
		}
		if appCount == 0 {
			return fmt.Errorf("批次至少需要包含一个应用")
		}

		// 7. 保存批次更新
		if err := tx.Save(batch).Error; err != nil {
			return fmt.Errorf("更新批次失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	logger.Info("批次更新成功",
		zap.String("batch_number", batch.BatchNumber),
		zap.Int64("batch_id", batch.ID),
		zap.Any("updated_fields", updatedFields))

	return batch, updatedFields, nil
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

// UpdateReleaseDependencies 更新批次应用的临时依赖配置
func (s *BatchService) UpdateReleaseDependencies(req *dto.UpdateReleaseDependenciesRequest) (*dto.ReleaseDependenciesResponse, error) {
	release, err := s.batchRepo.GetReleaseAppByID(req.ReleaseAppID)
	if err != nil {
		return nil, fmt.Errorf("发布应用不存在: %w", err)
	}

	if release.BatchID != req.BatchID {
		return nil, fmt.Errorf("发布应用不属于指定批次")
	}

	batch, err := s.batchRepo.GetByID(req.BatchID)
	if err != nil {
		return nil, fmt.Errorf("批次不存在: %w", err)
	}

	if batch.Status >= constants.BatchStatusSealed {
		return nil, fmt.Errorf("批次已封板或进入发布阶段，无法修改依赖")
	}

	if release.IsLocked {
		return nil, fmt.Errorf("发布记录已锁定，无法修改依赖")
	}

	normalizedTemp := normalizeDependencyIDs(req.TempDependsOn)

	var batchAppIDs []int64
	if err := s.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", req.BatchID).
		Pluck("app_id", &batchAppIDs).Error; err != nil {
		return nil, fmt.Errorf("查询批次应用失败: %w", err)
	}

	appIDSet := make(map[int64]struct{}, len(batchAppIDs))
	for _, id := range batchAppIDs {
		appIDSet[id] = struct{}{}
	}

	for _, depID := range normalizedTemp {
		if depID == release.AppID {
			return nil, fmt.Errorf("应用不能依赖自身")
		}
		if _, ok := appIDSet[depID]; !ok {
			return nil, fmt.Errorf("依赖的应用 %d 不在当前批次中", depID)
		}
	}

	type dependencyRow struct {
		AppID            int64
		TempDependsOn    []byte
		DefaultDependsOn []byte
	}

	var rows []dependencyRow
	if err := s.db.Table("release_apps").
		Select("release_apps.app_id as app_id, release_apps.temp_depends_on, applications.default_depends_on").
		Joins("JOIN applications ON release_apps.app_id = applications.id").
		Where("release_apps.batch_id = ?", req.BatchID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询依赖信息失败: %w", err)
	}

	graph := make(map[int64][]int64, len(rows))
	for _, row := range rows {
		defaultDeps, err := decodeDependencyIDs(row.DefaultDependsOn)
		if err != nil {
			return nil, fmt.Errorf("解析应用 %d 默认依赖失败: %w", row.AppID, err)
		}

		tempDeps, err := decodeDependencyIDs(row.TempDependsOn)
		if err != nil {
			return nil, fmt.Errorf("解析应用 %d 临时依赖失败: %w", row.AppID, err)
		}

		if row.AppID == release.AppID {
			tempDeps = normalizedTemp
		}

		graph[row.AppID] = filterDependenciesForBatch(defaultDeps, tempDeps, appIDSet)
	}

	if hasDependencyCycle(graph) {
		return nil, fmt.Errorf("依赖配置存在循环，请调整")
	}

	data, err := json.Marshal(normalizedTemp)
	if err != nil {
		return nil, fmt.Errorf("序列化依赖失败: %w", err)
	}

	now := time.Now()
	updates := map[string]any{
		"temp_depends_on": datatypes.JSON(data),
		"updated_at":      now,
	}

	if err := s.db.Model(&model.ReleaseApp{}).
		Where("id = ?", release.ID).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新临时依赖失败: %w", err)
	}

	defaultDeps := []int64{}
	if release.Application != nil {
		if deps, err := decodeDependencyIDs(release.Application.DefaultDependsOn); err == nil {
			defaultDeps = normalizeDependencyIDs(deps)
		} else {
			logger.Error("解析应用默认依赖失败",
				zap.Int64("app_id", release.Application.ID),
				zap.Error(err))
		}
	}

	logger.Info("更新发布应用临时依赖成功",
		zap.Int64("batch_id", req.BatchID),
		zap.Int64("release_app_id", release.ID),
		zap.Int64("app_id", release.AppID),
		zap.Any("temp_depends_on", normalizedTemp),
		zap.String("operator", req.Operator))

	return &dto.ReleaseDependenciesResponse{
		BatchID:          req.BatchID,
		ReleaseAppID:     release.ID,
		AppID:            release.AppID,
		DefaultDependsOn: defaultDeps,
		TempDependsOn:    normalizedTemp,
		UpdatedAt:        now.Format(time.RFC3339),
	}, nil
}

// DeleteBatch 删除批次
func (s *BatchService) DeleteBatch(batchID int64, operator string) error {
	// 1. 获取批次
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return fmt.Errorf("批次不存在: %w", err)
	}

	// 2. 检查批次状态（只能删除草稿批次）
	if batch.Status != constants.BatchStatusDraft {
		return fmt.Errorf("只能删除草稿状态的批次")
	}

	// 3. 使用事务删除
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 删除应用发布记录
		if err := tx.Where("batch_id = ?", batchID).Delete(&model.ReleaseApp{}).Error; err != nil {
			return fmt.Errorf("删除应用发布记录失败: %w", err)
		}

		// 删除批次
		if err := tx.Delete(&model.Batch{}, batchID).Error; err != nil {
			return fmt.Errorf("删除批次失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	logger.Info("批次删除成功",
		zap.Int64("batch_id", batchID),
		zap.String("operator", operator))

	return nil
}

// GetBatch 获取批次详情（返回 DTO，支持应用列表分页）
func (s *BatchService) GetBatch(batchID int64, appPage, appPageSize int) (*dto.BatchDetailResponse, error) {
	// 1. 获取批次基本信息
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, fmt.Errorf("批次不存在: %w", err)
	}

	// 2. 分页获取应用列表
	apps, totalApps, err := s.batchRepo.GetReleaseAppsByBatchID(batchID, appPage, appPageSize)
	if err != nil {
		return nil, fmt.Errorf("获取应用列表失败: %w", err)
	}

	// 3. 转换为响应格式（包含构建记录）
	appResponses := s.toReleaseAppResponses(apps)

	// 4. 构建详情响应
	response := &dto.BatchDetailResponse{
		BatchResponse: s.toBatchResponse(batch, totalApps),
		Apps:          appResponses,
		TotalApps:     totalApps,
		AppPage:       appPage,
		AppPageSize:   appPageSize,
	}

	appTypeConfigs := config.GetAppTypeConfigs()
	if len(appTypeConfigs) > 0 {
		response.AppTypeConfigs = make(map[string]dto.AppTypeConfigInfo, len(appTypeConfigs))
		for key, cfg := range appTypeConfigs {
			label := cfg.Label
			if label == "" {
				label = key
			}

			deps := make([]string, 0, len(cfg.Dependencies))
			deps = append(deps, cfg.Dependencies...)

			response.AppTypeConfigs[key] = dto.AppTypeConfigInfo{
				Label:        label,
				Description:  cfg.Description,
				Icon:         cfg.Icon,
				Color:        cfg.Color,
				Dependencies: deps,
			}
		}
	}

	return response, nil
}

// toReleaseAppResponses 转换 ReleaseApp 列表为 DTO（包含自上次部署以来的构建记录）
func (s *BatchService) toReleaseAppResponses(releases []*model.ReleaseApp) []dto.ReleaseAppResponse {
	responses := make([]dto.ReleaseAppResponse, len(releases))

	for i, release := range releases {
		releaseResp := dto.ReleaseAppResponse{
			// ReleaseApp 基本信息
			ID:      release.ID,
			BatchID: release.BatchID,
			AppID:   release.AppID,
			BuildID: release.BuildID,

			// 版本信息
			PreviousDeployedTag: release.PreviousDeployedTag,
			TargetTag:           release.TargetTag,

			// 发布信息
			ReleaseNotes: release.ReleaseNotes,
			IsLocked:     release.IsLocked,
			Reason:       release.Reason,
			Status:       release.Status,

			// 时间信息
			CreatedAt: release.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: release.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		releaseResp.DefaultDependsOn = []int64{}
		releaseResp.TempDependsOn = []int64{}

		// 填充构建信息（如果通过 Preload("Build") 已加载）
		if release.Build != nil {
			releaseResp.BuildNumber = &release.Build.BuildNumber
			releaseResp.BuildStatus = &release.Build.BuildStatus
			buildTime := release.Build.BuildCreated.Format("2006-01-02T15:04:05Z07:00")
			releaseResp.BuildTime = &buildTime
			releaseResp.ImageURL = &release.Build.ImageURL
			releaseResp.CommitSHA = &release.Build.CommitSHA
			releaseResp.CommitMessage = &release.Build.CommitMessage
			releaseResp.CommitBranch = &release.Build.CommitBranch
		}

		// 填充应用信息（如果已加载）
		if release.Application != nil {
			releaseResp.AppName = release.Application.Name
			releaseResp.AppDisplayName = release.Application.DisplayName
			releaseResp.AppType = release.Application.AppType
			releaseResp.AppProject = release.Application.Project
			releaseResp.AppStatus = release.Application.Status
			releaseResp.TeamID = release.Application.TeamID
			releaseResp.DeployedTag = release.Application.DeployedTag // 当前部署的标签

			if deps, err := decodeDependencyIDs(release.Application.DefaultDependsOn); err != nil {
				logger.Error("解析应用默认依赖失败",
					zap.Int64("app_id", release.Application.ID),
					zap.Error(err))
			} else {
				releaseResp.DefaultDependsOn = normalizeDependencyIDs(deps)
			}

			// 填充仓库信息
			if release.Application.Repository != nil {
				releaseResp.RepoID = release.Application.RepoID
				releaseResp.RepoName = release.Application.Repository.Name
				releaseResp.RepoFullName = release.Application.Repository.Project + "/" + release.Application.Repository.Name
			} else {
				releaseResp.RepoID = release.Application.RepoID
				releaseResp.RepoName = ""
				releaseResp.RepoFullName = ""
			}

			// 填充团队信息
			if release.Application.Team != nil {
				releaseResp.TeamName = &release.Application.Team.Name
			}

			// 【新增】填充最近的构建记录
			releaseResp.RecentBuilds = s.getRecentBuilds(release)
		} else {
			// 如果应用信息未加载，设置默认值
			releaseResp.AppName = ""
			releaseResp.AppType = ""
			releaseResp.AppProject = ""
			releaseResp.AppStatus = 0
			releaseResp.RepoID = 0
			releaseResp.RepoName = ""
			releaseResp.RepoFullName = ""
		}

		if deps, err := decodeDependencyIDs(release.TempDependsOn); err != nil {
			logger.Error("解析临时依赖失败",
				zap.Int64("release_id", release.ID),
				zap.Error(err))
		} else {
			releaseResp.TempDependsOn = normalizeDependencyIDs(deps)
		}

		responses[i] = releaseResp
	}

	return responses
}

// getRecentBuilds 获取应用最近的构建记录（方案A：基于 deployed_tag，自上次部署以来）
func (s *BatchService) getRecentBuilds(app *model.ReleaseApp) []dto.BuildSummary {
	const buildLimit = 15 // 固定返回15条

	var builds []*model.Build
	var err error

	// 1. 尝试基于 Application.DeployedTag 查找基准时间
	if app.Application != nil && app.Application.DeployedTag != nil {
		// 查找 deployed_tag 对应的构建记录
		deployedBuild, err := s.batchRepo.GetBuildByAppIDAndTag(app.AppID, *app.Application.DeployedTag)
		if err != nil {
			logger.Error("查询deployed_tag对应的构建失败",
				zap.Int64("app_id", app.AppID),
				zap.String("deployed_tag", *app.Application.DeployedTag),
				zap.Error(err))
			// 出错则返回空数组
			return []dto.BuildSummary{}
		}

		if deployedBuild != nil {
			// 找到了基准构建，查询该时间之后的构建
			builds, err = s.batchRepo.GetBuildsSinceTime(app.AppID, deployedBuild.BuildCreated, buildLimit)
			if err != nil {
				logger.Error("查询时间后的构建失败",
					zap.Int64("app_id", app.AppID),
					zap.Time("since_time", deployedBuild.BuildCreated),
					zap.Error(err))
				return []dto.BuildSummary{}
			}
		} else {
			// deployed_tag 对应的构建不存在，fallback 到最近15条
			logger.Warn("deployed_tag对应的构建不存在，返回最近15条",
				zap.Int64("app_id", app.AppID),
				zap.String("deployed_tag", *app.Application.DeployedTag))
			builds, err = s.batchRepo.GetRecentBuilds(app.AppID, buildLimit)
			if err != nil {
				logger.Error("查询最近构建失败",
					zap.Int64("app_id", app.AppID),
					zap.Error(err))
				return []dto.BuildSummary{}
			}
		}
	} else {
		// 2. 没有 deployed_tag（新应用），返回最近15条
		builds, err = s.batchRepo.GetRecentBuilds(app.AppID, buildLimit)
		if err != nil {
			logger.Error("查询最近构建失败（新应用）",
				zap.Int64("app_id", app.AppID),
				zap.Error(err))
			return []dto.BuildSummary{}
		}
	}

	// 3. 确保当前选中的构建也在列表中（如果存在build_id）
	if app.BuildID != nil {
		// 检查当前build_id是否已在列表中
		currentBuildInList := false
		for _, build := range builds {
			if build.ID == *app.BuildID {
				currentBuildInList = true
				break
			}
		}

		// 如果不在列表中，单独查询并添加到列表开头
		if !currentBuildInList {
			var currentBuild model.Build
			if err := s.db.Where("id = ?", *app.BuildID).First(&currentBuild).Error; err == nil {
				// 将当前构建添加到列表开头
				builds = append([]*model.Build{&currentBuild}, builds...)
			}
		}
	}

	// 4. 转换为 DTO
	return s.toBuildSummaries(builds)
}

// toBuildSummaries 转换构建记录为摘要格式
func (s *BatchService) toBuildSummaries(builds []*model.Build) []dto.BuildSummary {
	if len(builds) == 0 {
		return []dto.BuildSummary{} // 返回空数组而不是 nil
	}

	summaries := make([]dto.BuildSummary, len(builds))
	for i, build := range builds {
		summaries[i] = dto.BuildSummary{
			ID:            build.ID,
			BuildNumber:   build.BuildNumber,
			BuildStatus:   build.BuildStatus,
			ImageTag:      build.ImageTag,
			CommitSHA:     build.CommitSHA,
			CommitMessage: build.CommitMessage,
			CommitAuthor:  build.CommitAuthor,
			BuildCreated:  build.BuildCreated.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return summaries
}

func filterDependenciesForBatch(defaultDeps, tempDeps []int64, allowed map[int64]struct{}) []int64 {
	if len(allowed) == 0 {
		return []int64{}
	}

	combined := make([]int64, 0, len(defaultDeps)+len(tempDeps))
	for _, id := range defaultDeps {
		if _, ok := allowed[id]; ok {
			combined = append(combined, id)
		}
	}
	for _, id := range tempDeps {
		if _, ok := allowed[id]; ok {
			combined = append(combined, id)
		}
	}

	if len(combined) == 0 {
		return []int64{}
	}

	return normalizeDependencyIDs(combined)
}

// toBatchResponse 转换 Batch 模型为 BatchResponse DTO
func (s *BatchService) toBatchResponse(batch *model.Batch, appCount int64) dto.BatchResponse {
	response := dto.BatchResponse{
		// 基本信息
		ID:           batch.ID,
		BatchNumber:  batch.BatchNumber,
		Initiator:    batch.Initiator,
		ReleaseNotes: batch.ReleaseNotes,

		// 状态信息
		Status:         batch.Status,
		StatusName:     getStatusName(batch.Status),
		ApprovalStatus: batch.ApprovalStatus,
		AppCount:       appCount,

		// 审批信息
		ApprovedBy:   batch.ApprovedBy,
		ApprovedAt:   dto.FormatTime(batch.ApprovedAt),
		RejectReason: batch.RejectReason,

		// 时间追踪
		TaggedAt:             dto.FormatTime(batch.SealedAt),
		PreDeployStartedAt:   dto.FormatTime(batch.PreStartedAt),
		PreDeployFinishedAt:  dto.FormatTime(batch.PreFinishedAt),
		ProdDeployStartedAt:  dto.FormatTime(batch.ProdStartedAt),
		ProdDeployFinishedAt: dto.FormatTime(batch.ProdFinishedAt),

		// 验收信息
		FinalAcceptedAt: dto.FormatTime(batch.FinalAcceptedAt),
		FinalAcceptedBy: batch.FinalAcceptedBy,

		// 取消信息
		CancelledAt:  dto.FormatTime(batch.CancelledAt),
		CancelledBy:  batch.CancelledBy,
		CancelReason: batch.CancelReason,

		// 系统字段
		CreatedAt: batch.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: batch.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return response
}

// getStatusName 获取状态名称
func getStatusName(status int8) string {
	switch status {
	case constants.BatchStatusDraft:
		return "草稿"
	case constants.BatchStatusSealed:
		return "已封板"
	case constants.BatchStatusPreDeploying:
		return "预发布部署中"
	case constants.BatchStatusPreDeployed:
		return "预发布已部署"
	case constants.BatchStatusProdDeploying:
		return "生产部署中"
	case constants.BatchStatusProdDeployed:
		return "生产已部署"
	case constants.BatchStatusCompleted:
		return "已完成"
	case constants.BatchStatusCancelled:
		return "已取消"
	default:
		return "未知状态"
	}
}

// BatchWithAppCount 批次及应用数量
type BatchWithAppCount struct {
	Batch    *model.Batch
	AppCount int64
}

// ListBatches 查询批次列表（返回 DTO）
func (s *BatchService) ListBatches(page, pageSize int, statuses []int8, initiator string, approvalStatus *string, createdAtStart, createdAtEnd *time.Time, keyword string) ([]dto.BatchResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 50
	}

	// 查询批次列表
	batches, total, err := s.batchRepo.List(page, pageSize, statuses, initiator, approvalStatus, createdAtStart, createdAtEnd, keyword)
	if err != nil {
		return nil, 0, err
	}

	// 为每个批次查询应用数量并转换为 DTO
	responses := make([]dto.BatchResponse, len(batches))
	for i, batch := range batches {
		appCount, err := s.batchRepo.GetAppCountByBatchID(batch.ID)
		if err != nil {
			// 如果查询失败，设置为0
			appCount = 0
		}
		responses[i] = s.toBatchResponse(batch, appCount)
	}

	return responses, total, nil
}

// HandleBuildNotify 处理构建通知
func (s *BatchService) HandleBuildNotify(req *BuildNotifyRequest) error {
	// 1. 查找或创建构建记录
	build, err := s.findOrCreateBuild(req)
	if err != nil {
		return fmt.Errorf("处理构建记录失败: %w", err)
	}

	// 2. 查找该应用在草稿/未封板批次中的记录
	releaseApps, err := s.batchRepo.GetReleaseAppsByAppIDAndNotSealed(req.AppID)
	if err != nil {
		return fmt.Errorf("查询发布应用记录失败: %w", err)
	}

	if len(releaseApps) == 0 {
		logger.Info("应用不在任何未封板批次中，无需更新", zap.Int64("app_id", req.AppID), zap.String("tag", req.Tag))
		return nil
	}

	// 3. 更新所有未封板批次中的该应用记录（仅更新 build_id）
	for _, app := range releaseApps {
		updates := map[string]interface{}{
			"build_id":   build.ID,
			"target_tag": build.ImageTag,
		}

		if err := s.db.Model(&model.ReleaseApp{}).
			Where("id = ?", app.ID).
			Updates(updates).Error; err != nil {
			logger.Error("更新发布应用构建失败",
				zap.Int64("release_app_id", app.ID),
				zap.Error(err))
			continue
		}

		logger.Info("更新发布应用构建成功",
			zap.Int64("batch_id", app.BatchID),
			zap.Int64("app_id", app.AppID),
			zap.Int64("new_build_id", build.ID),
			zap.String("build_image_tag", build.ImageTag))
	}

	return nil
}

// BuildNotifyRequest 构建通知请求
type BuildNotifyRequest struct {
	RepoID        int64   `json:"repo_id" binding:"required"`
	AppID         int64   `json:"app_id" binding:"required"`
	BuildNumber   string  `json:"build_number" binding:"required"`
	Tag           string  `json:"tag" binding:"required"`
	CommitID      string  `json:"commit_id" binding:"required"`
	CommitMessage *string `json:"commit_message"`
	CommitAuthor  *string `json:"commit_author"`
	ImageName     string  `json:"image_name" binding:"required"`
	ImageTag      string  `json:"image_tag" binding:"required"`
	BuildStatus   string  `json:"build_status" binding:"required"`
}

// findOrCreateBuild 查找或创建构建记录
func (s *BatchService) findOrCreateBuild(req *BuildNotifyRequest) (*model.Build, error) {
	// 尝试查找已存在的构建
	var build model.Build
	err := s.db.Where("build_number = ?", req.BuildNumber).First(&build).Error

	if err == nil {
		// 构建记录已存在，更新状态
		build.BuildStatus = req.BuildStatus
		build.ImageURL = req.ImageName
		build.ImageTag = req.ImageTag
		build.CommitSHA = req.CommitID
		build.CommitMessage = *req.CommitMessage
		build.CommitAuthor = *req.CommitAuthor

		now := time.Now()
		if req.BuildStatus == "success" || req.BuildStatus == "failed" {
			build.BuildFinished = now
		}

		if err := s.db.Save(&build).Error; err != nil {
			return nil, err
		}

		return &build, nil
	}

	// 构建记录不存在，创建新记录
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 将 string 转为 int
	buildNum := 0
	fmt.Sscanf(req.BuildNumber, "%d", &buildNum)

	commitMsg := ""
	if req.CommitMessage != nil {
		commitMsg = *req.CommitMessage
	}
	commitAuthor := ""
	if req.CommitAuthor != nil {
		commitAuthor = *req.CommitAuthor
	}

	now := time.Now()
	build = model.Build{
		BuildNumber:   buildNum,
		AppID:         req.AppID,
		RepoID:        req.RepoID,
		ImageTag:      req.Tag,
		CommitSHA:     req.CommitID,
		CommitMessage: commitMsg,
		CommitAuthor:  commitAuthor,
		ImageURL:      req.ImageName,
		BuildStatus:   req.BuildStatus,
		BuildEvent:    "tag",
		BuildCreated:  now,
		BuildStarted:  now,
		BuildFinished: now,
	}

	if err := s.db.Create(&build).Error; err != nil {
		return nil, err
	}

	logger.Info("创建构建记录",
		zap.Int("build_number", build.BuildNumber),
		zap.Int64("build_id", build.ID))

	return &build, nil
}

// ============== 自定义错误类型 ==============

// AppConflictError 应用冲突错误
type AppConflictError struct {
	Conflicts map[int64]*model.Batch
	AppMap    map[int64]*model.Application
}

func (e *AppConflictError) Error() string {
	return "存在应用冲突"
}

// BatchSealedError 批次已封板错误
type BatchSealedError struct {
	BatchID int64
	Status  int8
}

func (e *BatchSealedError) Error() string {
	return fmt.Sprintf("批次已封板，不允许修改（状态: %d）", e.Status)
}

// ApproveBatch 审批通过批次（独立于 status 流转）
func (s *BatchService) ApproveBatch(batchID int64, operator string, reason string) error {
	var batch model.Batch
	if err := s.db.First(&batch, batchID).Error; err != nil {
		return fmt.Errorf("查询批次失败: %w", err)
	}

	// 检查审批状态
	if batch.ApprovalStatus == constants.ApprovalStatusApproved {
		return fmt.Errorf("批次已审批通过")
	}
	if batch.ApprovalStatus == constants.ApprovalStatusRejected {
		return fmt.Errorf("批次已被拒绝，不能再次审批")
	}

	// 更新审批状态
	now := time.Now()
	updates := map[string]interface{}{
		"approval_status": constants.ApprovalStatusApproved,
		"approved_by":     operator,
		"approved_at":     now,
	}

	if err := s.db.Model(&batch).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新审批状态失败: %w", err)
	}

	logger.Info("批次审批通过",
		zap.Int64("batch_id", batchID),
		zap.String("batch_number", batch.BatchNumber),
		zap.String("operator", operator))

	return nil
}

// RejectBatch 拒绝批次（独立于 status 流转）
func (s *BatchService) RejectBatch(batchID int64, operator string, reason string) error {
	var batch model.Batch
	if err := s.db.First(&batch, batchID).Error; err != nil {
		return fmt.Errorf("查询批次失败: %w", err)
	}

	// 检查审批状态
	if batch.ApprovalStatus == constants.ApprovalStatusApproved {
		return fmt.Errorf("批次已审批通过，不能拒绝")
	}
	if batch.ApprovalStatus == constants.ApprovalStatusRejected {
		return fmt.Errorf("批次已被拒绝")
	}

	// 更新审批状态
	updates := map[string]interface{}{
		"approval_status": constants.ApprovalStatusRejected,
		"reject_reason":   reason,
	}

	if err := s.db.Model(&batch).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新审批状态失败: %w", err)
	}

	logger.Info("批次已拒绝",
		zap.Int64("batch_id", batchID),
		zap.String("batch_number", batch.BatchNumber),
		zap.String("operator", operator),
		zap.String("reason", reason))

	return nil
}

// GetBatchStatus 获取批次状态（轻量级，用于状态轮询）
// 只查询 release_batches 和 release_apps 两个表，不关联其他表
func (s *BatchService) GetBatchStatus(batchID int64, appPage, appPageSize int) (*dto.BatchStatusResponse, error) {
	// 1. 查询批次基本信息
	var batch model.Batch
	if err := s.db.First(&batch, batchID).Error; err != nil {
		return nil, fmt.Errorf("查询批次失败: %w", err)
	}

	// 2. 查询应用总数
	var totalApps int64
	if err := s.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batchID).
		Count(&totalApps).Error; err != nil {
		return nil, fmt.Errorf("查询应用总数失败: %w", err)
	}

	// 3. 查询应用状态列表（分页）
	var releaseApps []model.ReleaseApp
	offset := (appPage - 1) * appPageSize
	if err := s.db.Where("batch_id = ?", batchID).
		Order("id ASC").
		Limit(appPageSize).
		Offset(offset).
		Find(&releaseApps).Error; err != nil {
		return nil, fmt.Errorf("查询应用状态失败: %w", err)
	}

	// 4. 转换为响应格式
	apps := make([]dto.ReleaseAppStatusResponse, len(releaseApps))
	for i, app := range releaseApps {
		apps[i] = dto.ReleaseAppStatusResponse{
			ID:       app.ID,
			AppID:    app.AppID,
			Status:   app.Status,
			IsLocked: app.IsLocked,
		}
	}

	// 5. 获取状态名称
	statusName := getStatusName(batch.Status)

	// 6. 构造响应
	response := &dto.BatchStatusResponse{
		ID:             batch.ID,
		BatchNumber:    batch.BatchNumber,
		Status:         batch.Status,
		StatusName:     statusName,
		ApprovalStatus: batch.ApprovalStatus,

		SealedAt:             dto.FormatTime(batch.SealedAt),
		PreDeployStartedAt:   dto.FormatTime(batch.PreStartedAt),
		PreDeployFinishedAt:  dto.FormatTime(batch.PreFinishedAt),
		ProdDeployStartedAt:  dto.FormatTime(batch.ProdStartedAt),
		ProdDeployFinishedAt: dto.FormatTime(batch.ProdFinishedAt),
		FinalAcceptedAt:      dto.FormatTime(batch.FinalAcceptedAt),
		CancelledAt:          dto.FormatTime(batch.CancelledAt),
		UpdatedAt:            batch.UpdatedAt.Format(time.RFC3339),

		Apps:        apps,
		TotalApps:   totalApps,
		AppPage:     appPage,
		AppPageSize: appPageSize,
	}

	return response, nil
}
