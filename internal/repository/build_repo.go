package repository

import (
	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/errors"

	"gorm.io/gorm"
)

// BuildRepository 构建记录仓储接口
type BuildRepository interface {
	Create(build *model.Build) error
	FindByID(id int64) (*model.Build, error)
	FindByAppAndNumber(appID int64, buildNumber int) (*model.Build, error)
	List(page, pageSize int, repoID, appID *int64, buildStatus, buildEvent, imageTag, commitSHA, environment *string, keyword string) ([]*model.Build, int64, error)
	ListByRepoID(repoID int64, limit int) ([]*model.Build, error)
	ListByAppID(appID int64, limit int) ([]*model.Build, error)
	Update(build *model.Build) error
	Delete(id int64) error
}

type buildRepository struct {
	db *gorm.DB
}

// NewBuildRepository 创建构建记录仓储实例
func NewBuildRepository(db *gorm.DB) BuildRepository {
	return &buildRepository{db: db}
}

// Create 创建构建记录
func (r *buildRepository) Create(build *model.Build) error {
	if err := r.db.Create(build).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建构建记录失败", err)
	}
	return nil
}

// FindByID 根据ID查询构建记录
func (r *buildRepository) FindByID(id int64) (*model.Build, error) {
	var build model.Build
	err := r.db.Preload("Repository").Preload("Application").
		First(&build, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询构建记录失败", err)
	}
	return &build, nil
}

// FindByAppAndNumber 根据应用ID和构建号查询
func (r *buildRepository) FindByAppAndNumber(appID int64, buildNumber int) (*model.Build, error) {
	var build model.Build
	err := r.db.Preload("Repository").Preload("Application").
		Where("app_id = ? AND build_number = ?", appID, buildNumber).
		First(&build).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询构建记录失败", err)
	}
	return &build, nil
}

// List 分页查询构建记录列表
func (r *buildRepository) List(page, pageSize int, repoID, appID *int64,
	buildStatus, buildEvent, imageTag, commitSHA, environment *string, keyword string) ([]*model.Build, int64, error) {

	var builds []*model.Build
	var total int64

	query := r.db.Model(&model.Build{}).Preload("Repository").Preload("Application")

	// 按仓库筛选
	if repoID != nil {
		query = query.Where("repo_id = ?", *repoID)
	}

	// 按应用筛选
	if appID != nil {
		query = query.Where("app_id = ?", *appID)
	}

	// 按状态筛选
	if buildStatus != nil && *buildStatus != "" {
		query = query.Where("build_status = ?", *buildStatus)
	}

	// 按事件筛选
	if buildEvent != nil && *buildEvent != "" {
		query = query.Where("build_event = ?", *buildEvent)
	}

	// 按镜像标签筛选
	if imageTag != nil && *imageTag != "" {
		query = query.Where("image_tag = ?", *imageTag)
	}

	// 按 commit SHA 筛选
	if commitSHA != nil && *commitSHA != "" {
		query = query.Where("commit_sha LIKE ?", *commitSHA+"%")
	}

	// 按环境筛选
	if environment != nil && *environment != "" {
		query = query.Where("environment = ?", *environment)
	}

	// 关键字搜索（搜索 commit_message, commit_author）
	if keyword != "" {
		query = query.Where("commit_message LIKE ? OR commit_author LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计构建记录失败", err)
	}

	// 分页查询，按创建时间倒序
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&builds).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询构建记录列表失败", err)
	}

	return builds, total, nil
}

// ListByRepoID 查询某个仓库的最近构建记录
func (r *buildRepository) ListByRepoID(repoID int64, limit int) ([]*model.Build, error) {
	var builds []*model.Build
	err := r.db.Preload("Application").
		Where("repo_id = ?", repoID).
		Order("created_at DESC").
		Limit(limit).
		Find(&builds).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询仓库构建记录失败", err)
	}
	return builds, nil
}

// ListByAppID 查询某个应用的最近构建记录
func (r *buildRepository) ListByAppID(appID int64, limit int) ([]*model.Build, error) {
	var builds []*model.Build
	err := r.db.Where("app_id = ?", appID).
		Order("created_at DESC").
		Limit(limit).
		Find(&builds).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用构建记录失败", err)
	}
	return builds, nil
}

// Update 更新构建记录
func (r *buildRepository) Update(build *model.Build) error {
	if err := r.db.Save(build).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新构建记录失败", err)
	}
	return nil
}

// Delete 删除构建记录
func (r *buildRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.Build{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除构建记录失败", err)
	}
	return nil
}
