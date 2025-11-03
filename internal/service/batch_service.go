package service

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"devops-cd/internal/model"
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

// GetBatch 获取批次详情
func (s *BatchService) GetBatch(batchID int64) (*model.Batch, []*model.ReleaseApp, error) {
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, nil, fmt.Errorf("批次不存在: %w", err)
	}

	apps, err := s.batchRepo.GetReleaseAppsByBatchID(batchID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取应用列表失败: %w", err)
	}

	return batch, apps, nil
}

// BatchWithAppCount 批次及应用数量
type BatchWithAppCount struct {
	Batch    *model.Batch
	AppCount int64
}

// ListBatches 查询批次列表（返回带应用数量的批次）
func (s *BatchService) ListBatches(page, pageSize int, statuses []int8, initiator string, approvalStatus *string, createdAtStart, createdAtEnd *time.Time, keyword string) ([]*BatchWithAppCount, int64, error) {
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

	// 为每个批次查询应用数量
	result := make([]*BatchWithAppCount, len(batches))
	for i, batch := range batches {
		appCount, err := s.batchRepo.GetAppCountByBatchID(batch.ID)
		if err != nil {
			// 如果查询失败，设置为0
			appCount = 0
		}
		result[i] = &BatchWithAppCount{
			Batch:    batch,
			AppCount: appCount,
		}
	}

	return result, total, nil
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

		now := time.Now().Unix()
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

	now := time.Now().Unix()
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
