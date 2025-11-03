package repository

import (
	"time"

	"gorm.io/gorm"

	"devops-cd/internal/model"
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

// BatchWithCount 批次及应用数量
type BatchWithCount struct {
	Batch    *model.Batch
	AppCount int64
}

// List 分页查询批次列表
func (r *BatchRepository) List(page, pageSize int, statuses []int8, initiator string, approvalStatus *string, createdAtStart, createdAtEnd *time.Time, keyword string) ([]*model.Batch, int64, error) {
	var batches []*model.Batch
	var total int64

	query := r.db.Model(&model.Batch{})

	// 条件过滤
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if initiator != "" {
		query = query.Where("initiator = ?", initiator)
	}

	// 新增：审批状态过滤
	if approvalStatus != nil && *approvalStatus != "" {
		query = query.Where("approval_status = ?", *approvalStatus)
	}

	// 新增：时间范围过滤
	if createdAtStart != nil {
		query = query.Where("created_at >= ?", *createdAtStart)
	}
	if createdAtEnd != nil {
		query = query.Where("created_at <= ?", *createdAtEnd)
	}

	// 新增：关键字模糊搜索（批次编号、发起人、发布说明）
	if keyword != "" {
		query = query.Where(
			"batch_number LIKE ? OR initiator LIKE ? OR release_notes LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&batches).Error

	return batches, total, err
}

// GetAppCountByBatchID 获取批次的应用数量
func (r *BatchRepository) GetAppCountByBatchID(batchID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batchID).Count(&count).Error
	return count, err
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

// GetReleaseAppsByBatchID 获取批次的所有应用（包含应用详情、仓库信息和构建信息）
func (r *BatchRepository) GetReleaseAppsByBatchID(batchID int64) ([]*model.ReleaseApp, error) {
	var apps []*model.ReleaseApp
	err := r.db.Where("batch_id = ?", batchID).
		Preload("Application").            // 加载应用信息
		Preload("Application.Repository"). // 加载仓库信息
		Preload("Application.Team").       // 加载团队信息
		Preload("Build").                  // 加载构建信息（通过 build_id）
		Order("created_at ASC").           // 按创建时间排序
		Find(&apps).Error
	return apps, err
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
			conflicts[r.AppID] = &model.Batch{
				ID:          r.BatchID,
				BatchNumber: r.BatchNumber,
				Status:      r.Status,
			}
		}
	}

	return conflicts, nil
}

// ================== Build 相关 ==================

// GetLatestSuccessBuilds 获取应用列表的最新成功构建
func (r *BatchRepository) GetLatestSuccessBuilds(appIDs []int64) (map[int64]*model.Build, error) {
	var builds []*model.Build

	// 使用子查询找到每个应用的最新成功构建
	subQuery := r.db.Model(&model.Build{}).
		Select("app_id, MAX(created_at) as max_created_at").
		Where("app_id IN ? AND build_status = ?", appIDs, "success").
		Group("app_id")

	err := r.db.Table("builds").
		Joins("JOIN (?) as latest ON builds.app_id = latest.app_id AND builds.created_at = latest.max_created_at", subQuery).
		Where("builds.app_id IN ?", appIDs).
		Find(&builds).Error

	if err != nil {
		return nil, err
	}

	// 转换为map
	buildMap := make(map[int64]*model.Build)
	for _, build := range builds {
		buildMap[build.AppID] = build
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
