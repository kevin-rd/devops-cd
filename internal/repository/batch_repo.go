package repository

import (
	"devops-cd/internal/dto"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"devops-cd/internal/model"
	"devops-cd/internal/pkg/logger"
	"devops-cd/pkg/constants"
)

// BatchRepository 批次数据访问层
type BatchRepository struct {
	db *gorm.DB
}

// NewBatchRepository 创建批次Repository
func NewBatchRepository(db *gorm.DB) *BatchRepository {
	return &BatchRepository{
		db: db,
	}
}

// Create 创建批次
func (r *BatchRepository) Create(batch *model.Batch) error {
	return r.db.Create(batch).Error
}

// GetByID 根据ID获取批次
func (r *BatchRepository) GetByID(id int64) (*model.Batch, error) {
	var batch model.Batch
	err := r.db.First(&batch, id).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// GetByBatchNumber 根据批次编号获取
func (r *BatchRepository) GetByBatchNumber(batchNumber string) (*model.Batch, error) {
	var batch model.Batch
	err := r.db.Where("batch_number = ?", batchNumber).First(&batch).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// Update 更新批次
func (r *BatchRepository) Update(batch *model.Batch) error {
	return r.db.Save(batch).Error
}

// Delete 删除批次
func (r *BatchRepository) Delete(id int64) error {
	return r.db.Delete(&model.Batch{}, id).Error
}

// List 分页查询批次列表
func (r *BatchRepository) List(req dto.BatchListParam) ([]*model.Batch, int64, error) {
	var batches []*model.Batch
	var total int64

	// Where 条件
	applyFilters := func(query *gorm.DB) *gorm.DB {
		// 条件过滤
		if len(req.Statuses) > 0 {
			query = query.Where("status IN ?", req.Statuses)
		}
		if req.Initiator != nil && *req.Initiator != "" {
			query = query.Where("initiator = ?", *req.Initiator)
		}

		// 新增：审批状态过滤
		if req.ApprovalStatus != nil && *req.ApprovalStatus != "" {
			query = query.Where("approval_status = ?", *req.ApprovalStatus)
		}

		// 新增：时间范围过滤
		if req.CreatedAtStart != nil {
			query = query.Where("release_batches.created_at >= ?", *req.CreatedAtStart)
		}
		if req.CreatedAtEnd != nil {
			query = query.Where("release_batches.created_at <= ?", *req.CreatedAtEnd)
		}

		// 新增：关键字模糊搜索（批次编号、发起人、发布说明）
		if req.Keyword != nil && *req.Keyword != "" {
			query = query.Where(
				"batch_number LIKE ? OR initiator LIKE ? OR release_notes LIKE ?",
				"%"+*req.Keyword+"%", "%"+*req.Keyword+"%", "%"+*req.Keyword+"%",
			)
		}
		return query
	}

	// 统计总数// 统计总数
	if err := applyFilters(r.db.Model(&model.Batch{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	err := applyFilters(r.db.Model(&model.Batch{})).
		Select(`release_batches.*, COALESCE(COUNT(release_apps.id), 0) AS apps_count`).
		Joins(`LEFT JOIN release_apps ON release_apps.batch_id = release_batches.id`).
		Group(`release_batches.created_at, release_batches.id`).
		Order("release_batches.created_at DESC").Limit(req.PageSize).Offset(offset).Scan(&batches).Error

	return batches, total, err
}

// ================== ReleaseApp 相关 ==================

// CreateReleaseApp 创建发布应用记录
func (r *BatchRepository) CreateReleaseApp(app *model.ReleaseApp) error {
	return r.db.Create(app).Error
}

// BatchCreateReleaseApps 批量创建发布应用记录
func (r *BatchRepository) BatchCreateReleaseApps(apps []*model.ReleaseApp) error {
	if len(apps) == 0 {
		return nil
	}
	return r.db.Create(&apps).Error
}

// GetReleaseAppByID 获取单个发布应用记录（包含应用基础信息）
func (r *BatchRepository) GetReleaseAppByID(id int64) (*model.ReleaseApp, error) {
	var release model.ReleaseApp
	if err := r.db.Preload("Application").First(&release, id).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

// GetReleaseAppsByBatchID 获取批次的所有应用（包含应用详情、仓库信息和构建信息，支持分页）
func (r *BatchRepository) GetReleaseAppsByBatchID(batchID int64, page, pageSize int) ([]*model.ReleaseApp, int64, error) {
	var apps []*model.ReleaseApp
	var total int64

	// 统计总数
	if err := r.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batchID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.Where("batch_id = ?", batchID).
		Preload("Application").            // 加载应用信息
		Preload("Application.Repository"). // 加载仓库信息
		Preload("Application.Team").       // 加载团队信息
		Preload("Build").                  // 加载构建信息（通过 build_id）
		Order("created_at ASC").           // 按创建时间排序
		Offset(offset).
		Limit(pageSize).
		Find(&apps).Error

	return apps, total, err
}

// GetBuildByAppIDAndTag 根据应用ID和镜像标签查询构建记录
func (r *BatchRepository) GetBuildByAppIDAndTag(appID int64, imageTag string) (*model.Build, error) {
	var build model.Build
	err := r.db.Where("app_id = ? AND image_tag = ?", appID, imageTag).
		First(&build).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 未找到，返回 nil 而不是错误
		}
		return nil, err
	}
	return &build, nil
}

// GetBuildsSinceTime 获取应用在指定时间之后的成功构建记录
func (r *BatchRepository) GetBuildsSinceTime(appID int64, sinceTime time.Time, limit int) ([]*model.Build, error) {
	var builds []*model.Build
	query := r.db.Where("app_id = ?", appID).
		Where("build_created > ?", sinceTime).
		Where("build_status = ?", "success"). // 只返回成功的构建
		Order("build_created DESC")           // 时间倒序（最新在前）

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&builds).Error
	return builds, err
}

// GetRecentBuilds 获取应用最近的成功构建记录（用于新应用）
func (r *BatchRepository) GetRecentBuilds(appID int64, limit int) ([]*model.Build, error) {
	var builds []*model.Build
	err := r.db.Where("app_id = ?", appID).
		Where("build_status = ?", "success"). // 只返回成功的构建
		Order("build_created DESC").          // 时间倒序（最新在前）
		Limit(limit).
		Find(&builds).Error
	return builds, err
}

// DeleteReleaseApp 删除发布应用记录
func (r *BatchRepository) DeleteReleaseApp(id int64) error {
	return r.db.Delete(&model.ReleaseApp{}, id).Error
}

// DeleteReleaseAppsByBatchID 删除批次的所有应用
func (r *BatchRepository) DeleteReleaseAppsByBatchID(batchID int64) error {
	return r.db.Where("batch_id = ?", batchID).Delete(&model.ReleaseApp{}).Error
}

// DeleteReleaseAppsByBatchIDAndAppIDs 删除指定批次的指定应用
func (r *BatchRepository) DeleteReleaseAppsByBatchIDAndAppIDs(batchID int64, appIDs []int64) error {
	return r.db.Where("batch_id = ? AND app_id IN ?", batchID, appIDs).
		Delete(&model.ReleaseApp{}).Error
}

// UpdateReleaseApp 更新发布应用记录
func (r *BatchRepository) UpdateReleaseApp(app *model.ReleaseApp) error {
	return r.db.Save(app).Error
}

// GetReleaseAppsByAppIDAndNotSealed 获取指定应用在未封板批次中的记录
func (r *BatchRepository) GetReleaseAppsByAppIDAndNotSealed(appID int64) ([]*model.ReleaseApp, error) {
	var apps []*model.ReleaseApp
	err := r.db.Joins("JOIN release_batches ON release_apps.batch_id = release_batches.id").
		Where("release_apps.app_id = ? AND release_batches.status < ?", appID, constants.BatchStatusSealed).
		Where("release_batches.status NOT IN ?", []int8{constants.BatchStatusCancelled}).
		Find(&apps).Error
	return apps, err
}

// CheckAppConflict 检查应用是否在进行中的批次里
func (r *BatchRepository) CheckAppConflict(appIDs []int64, excludeBatchID *int64) (map[int64]*model.Batch, error) {
	type Result struct {
		AppID       int64
		BatchID     int64
		BatchNumber string
		Status      int8
	}

	var results []Result
	query := r.db.Table("release_apps").
		Select("release_apps.app_id, release_batches.id as batch_id, release_batches.batch_number, release_batches.status").
		Joins("JOIN release_batches ON release_apps.batch_id = release_batches.id").
		Where("release_apps.app_id IN ?", appIDs).
		Where("release_batches.status < ?", constants.BatchStatusCompleted).
		Where("release_batches.status NOT IN ?", []int8{constants.BatchStatusCancelled})

	if excludeBatchID != nil {
		query = query.Where("release_batches.id != ?", *excludeBatchID)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	// 构建冲突map
	conflicts := make(map[int64]*model.Batch)
	for _, r := range results {
		if _, exists := conflicts[r.AppID]; !exists {
			b := &model.Batch{
				BatchNumber: r.BatchNumber,
				Status:      r.Status,
			}
			b.ID = r.BatchID
			conflicts[r.AppID] = b
		}
	}

	return conflicts, nil
}

// ================== Build 相关 ==================

// GetLatestBuildsAfterDeployment 获取应用列表在上次部署后的最新成功构建
// 如果应用有 deployed_tag，则获取该 tag 之后的最新构建
// 如果没有 deployed_tag 或查找失败，则 fallback 到最新成功构建
func (r *BatchRepository) GetLatestBuildsAfterDeployment(appIDs []int64) (map[int64]*model.Build, error) {
	// 1. 查询所有应用的信息（包括 deployed_tag）
	var apps []*model.Application
	if err := r.db.Where("id IN ?", appIDs).Find(&apps).Error; err != nil {
		return nil, err
	}

	buildMap := make(map[int64]*model.Build)

	// 2. 对每个应用单独处理
	for _, app := range apps {
		log := logger.Sugar().With(zap.Int64("app_id", app.ID))
		var build *model.Build

		// 如果应用有 deployed_tag，尝试获取该 tag 之后的最新构建
		if app.DeployedTag != nil && *app.DeployedTag != "" {
			// 2.1 查找 deployed_tag 对应的构建记录
			deployedBuild, err := r.GetBuildByAppIDAndTag(app.ID, *app.DeployedTag)
			if err != nil {
				// 查询失败，记录日志
				log.Warnf("查询app:%v deployed_tag:%v 对应的构建失败: %v", app.ID, *app.DeployedTag, err)
				continue
			} else if deployedBuild != nil {
				// 2.2 查找该构建之后的最新成功构建
				var buildsAfter []*model.Build
				if err := r.db.Where("app_id = ?", app.ID).
					Where("build_created > ?", deployedBuild.BuildCreated).
					Where("build_status = ?", "success").
					Order("build_created DESC").
					Limit(1).Find(&buildsAfter).Error; err == nil && len(buildsAfter) > 0 {
					build = buildsAfter[0]
				} else {
					// 没有找到部署后的新构建
					log.Debugf("deployed_tag: %s 之后没有新构建", *app.DeployedTag)
					continue
				}
			} else {
				// deployed_tag 对应的构建不存在
				log.Warnf("deployed_tag: %s 对应的构建不存在", *app.DeployedTag)
				continue
			}
		}

		// 3. Fallback：如果还没有找到构建，使用最新成功构建
		if build == nil {
			var latestBuilds []*model.Build
			err := r.db.Where("app_id = ?", app.ID).
				Where("build_status = ?", "success").
				Order("build_created DESC").
				Limit(1).
				Find(&latestBuilds).Error

			if err == nil && len(latestBuilds) > 0 {
				build = latestBuilds[0]
			}
		}

		// 4. 如果找到构建，添加到结果map中
		if build != nil {
			buildMap[app.ID] = build
		}
	}

	return buildMap, nil
}

// GetBuildByID 根据ID获取构建记录
func (r *BatchRepository) GetBuildByID(id int64) (*model.Build, error) {
	var build model.Build
	err := r.db.First(&build, id).Error
	if err != nil {
		return nil, err
	}
	return &build, nil
}

// CreateBuild 创建构建记录
func (r *BatchRepository) CreateBuild(build *model.Build) error {
	return r.db.Create(build).Error
}

// UpdateBuild 更新构建记录
func (r *BatchRepository) UpdateBuild(build *model.Build) error {
	return r.db.Save(build).Error
}

// ================== Application 相关 ==================

// GetApplicationByID 根据ID获取应用
func (r *BatchRepository) GetApplicationByID(id int64) (*model.Application, error) {
	var app model.Application
	err := r.db.First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetApplicationsByIDs 批量获取应用
func (r *BatchRepository) GetApplicationsByIDs(ids []int64) (map[int64]*model.Application, error) {
	var apps []*model.Application
	err := r.db.Where("id IN ?", ids).Find(&apps).Error
	if err != nil {
		return nil, err
	}

	appMap := make(map[int64]*model.Application)
	for _, app := range apps {
		appMap[app.ID] = app
	}

	return appMap, nil
}

// GetApplicationWithBatch 获取应用及其关联的批次信息
func (r *BatchRepository) GetApplicationWithBatch(batchID, appID int64) (*model.ReleaseApp, error) {
	var releaseApp model.ReleaseApp
	err := r.db.Where("batch_id = ? AND app_id = ?", batchID, appID).First(&releaseApp).Error
	if err != nil {
		return nil, err
	}
	return &releaseApp, nil
}
