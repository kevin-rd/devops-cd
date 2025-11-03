package repository

import (
	"gorm.io/gorm"

	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/errors"
)

type RepositoryRepository interface {
	Create(repo *model.Repository) error
	FindByID(id int64) (*model.Repository, error)
	FindByProjectAndName(project, name string) (*model.Repository, error)
	List(page, pageSize int, project *string, teamID *int64, gitType *string, keyword string, status *int8) ([]*model.Repository, int64, error)
	Update(repo *model.Repository) error
	Delete(id int64) error
}

type repositoryRepository struct {
	db *gorm.DB
}

func NewRepositoryRepository(db *gorm.DB) RepositoryRepository {
	return &repositoryRepository{db: db}
}

func (r *repositoryRepository) Create(repo *model.Repository) error {
	if err := r.db.Create(repo).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建代码库失败", err)
	}
	return nil
}

func (r *repositoryRepository) FindByID(id int64) (*model.Repository, error) {
	var repo model.Repository
	err := r.db.Preload("Team").First(&repo, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询代码库失败", err)
	}
	return &repo, nil
}

func (r *repositoryRepository) FindByProjectAndName(project, name string) (*model.Repository, error) {
	var repo model.Repository
	err := r.db.Where("project = ? AND name = ? AND deleted_at IS NULL", project, name).First(&repo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询代码库失败", err)
	}
	return &repo, nil
}

func (r *repositoryRepository) List(page, pageSize int, project *string, teamID *int64, gitType *string, keyword string, status *int8) ([]*model.Repository, int64, error) {
	var repos []*model.Repository
	var total int64

	query := r.db.Model(&model.Repository{}).Preload("Team")

	// 过滤条件
	if project != nil {
		query = query.Where("project = ?", *project)
	}
	if teamID != nil {
		query = query.Where("team_id = ?", *teamID)
	}
	if gitType != nil {
		query = query.Where("git_type = ?", *gitType)
	}
	if keyword != "" {
		query = query.Where("project LIKE ? OR name LIKE ? OR description LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计代码库数量失败", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&repos).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询代码库列表失败", err)
	}

	return repos, total, nil
}

func (r *repositoryRepository) Update(repo *model.Repository) error {
	if err := r.db.Save(repo).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新代码库失败", err)
	}
	return nil
}

func (r *repositoryRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.Repository{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除代码库失败", err)
	}
	return nil
}
