package repository

import (
	"gorm.io/gorm"

	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/errors"
)

type RepositoryRepository interface {
	Create(repo *model.Repository) error
	FindByID(id int64) (*model.Repository, error)
	FindByNamespaceAndName(namespace, name string) (*model.Repository, error)
	List(page, pageSize int, namespace *string, teamID *int64, gitType *string, keyword string, status *int8) ([]*model.Repository, int64, error)
	Update(repo *model.Repository) error
	Delete(id int64) error
	Upsert(repo *model.Repository) error // 新增: 插入或更新
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
	err := r.db.Preload("Team").Preload("Project").First(&repo, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询代码库失败", err)
	}
	return &repo, nil
}

func (r *repositoryRepository) FindByNamespaceAndName(namespace, name string) (*model.Repository, error) {
	var repo model.Repository
	err := r.db.Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).First(&repo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询代码库失败", err)
	}
	return &repo, nil
}

func (r *repositoryRepository) List(page, pageSize int, namespace *string, teamID *int64, gitType *string, keyword string, status *int8) ([]*model.Repository, int64, error) {
	var repos []*model.Repository
	var total int64

	query := r.db.Model(&model.Repository{}).Preload("Team").Preload("Project")

	// 过滤条件
	if namespace != nil {
		query = query.Where("namespace = ?", *namespace)
	}
	if teamID != nil {
		query = query.Where("team_id = ?", *teamID)
	}
	if gitType != nil {
		query = query.Where("git_type = ?", *gitType)
	}
	if keyword != "" {
		query = query.Where("namespace LIKE ? OR name LIKE ? OR description LIKE ?",
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

// Upsert 插入或更新代码库（基于namespace+name唯一索引）
func (r *repositoryRepository) Upsert(repo *model.Repository) error {
	// 先尝试查找现有记录
	existing, err := r.FindByNamespaceAndName(repo.Namespace, repo.Name)

	if err != nil && err != pkgErrors.ErrRecordNotFound {
		return err
	}

	if existing != nil {
		// 更新现有记录
		repo.ID = existing.ID
		// 保留某些字段不被覆盖（如果需要）
		// 例如：team_id 可能是手动设置的，不应该被自动同步覆盖
		if repo.TeamID == nil {
			repo.TeamID = existing.TeamID
		}
		err = r.db.Updates(&repo).Error
	} else {
		// 插入新记录
		err = r.db.Create(&repo).Error
	}

	if err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "插入或更新代码库失败", err)
	}
	return nil
}
