package repository

import (
	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/responses"

	"gorm.io/gorm"
)

type ProjectEnvConfigRepository interface {
	Create(config *model.ProjectEnvConfig) error
	Update(config *model.ProjectEnvConfig) error
	DeleteByID(id int64) error
	DeleteByProjectID(projectID int64) error
	DeleteByProjectIDAndEnv(projectID int64, env string) error
	FindByID(id int64) (*model.ProjectEnvConfig, error)
	FindByProjectID(projectID int64) ([]*model.ProjectEnvConfig, error)
	FindByProjectIDAndEnv(projectID int64, env string) (*model.ProjectEnvConfig, error)
	BatchCreate(configs []*model.ProjectEnvConfig) error
}

type projectEnvConfigRepository struct {
	db *gorm.DB
}

func NewProjectEnvConfigRepository(db *gorm.DB) ProjectEnvConfigRepository {
	return &projectEnvConfigRepository{db: db}
}

func (r *projectEnvConfigRepository) Create(config *model.ProjectEnvConfig) error {
	if err := r.db.Create(config).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建项目环境配置失败", err)
	}
	return nil
}

func (r *projectEnvConfigRepository) Update(config *model.ProjectEnvConfig) error {
	if err := r.db.Save(config).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新项目环境配置失败", err)
	}
	return nil
}

func (r *projectEnvConfigRepository) DeleteByID(id int64) error {
	if err := r.db.Where("id = ?", id).Delete(&model.ProjectEnvConfig{}).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除项目环境配置失败", err)
	}
	return nil
}

func (r *projectEnvConfigRepository) DeleteByProjectID(projectID int64) error {
	if err := r.db.Where("project_id = ?", projectID).Delete(&model.ProjectEnvConfig{}).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除项目环境配置失败", err)
	}
	return nil
}

func (r *projectEnvConfigRepository) DeleteByProjectIDAndEnv(projectID int64, env string) error {
	if err := r.db.Where("project_id = ? AND env = ?", projectID, env).Delete(&model.ProjectEnvConfig{}).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除项目环境配置失败", err)
	}
	return nil
}

func (r *projectEnvConfigRepository) FindByID(id int64) (*model.ProjectEnvConfig, error) {
	var config model.ProjectEnvConfig
	err := r.db.Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目环境配置失败", err)
	}
	return &config, nil
}

func (r *projectEnvConfigRepository) FindByProjectID(projectID int64) ([]*model.ProjectEnvConfig, error) {
	var configs []*model.ProjectEnvConfig
	if err := r.db.Where("project_id = ?", projectID).Order("env ASC").Find(&configs).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目环境配置列表失败", err)
	}
	return configs, nil
}

func (r *projectEnvConfigRepository) FindByProjectIDAndEnv(projectID int64, env string) (*model.ProjectEnvConfig, error) {
	var config model.ProjectEnvConfig
	err := r.db.Where("project_id = ? AND env = ?", projectID, env).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目环境配置失败", err)
	}
	return &config, nil
}

func (r *projectEnvConfigRepository) BatchCreate(configs []*model.ProjectEnvConfig) error {
	if len(configs) == 0 {
		return nil
	}
	if err := r.db.Create(&configs).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "批量创建项目环境配置失败", err)
	}
	return nil
}
