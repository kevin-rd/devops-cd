package repository

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	pkgErrors "devops-cd/pkg/responses"

	"gorm.io/gorm"
)

type AppEnvConfigRepository interface {
	Create(config *model.AppEnvConfig) error
	Update(config *model.AppEnvConfig) error
	Delete(id int64) error
	FindByID(id int64) (*model.AppEnvConfig, error)
	FindByAppID(appID int64) ([]*model.AppEnvConfig, error)
	FindByAppIDAndEnv(appID int64, env string) ([]*model.AppEnvConfig, error)
	CheckExists(appID int64, env, cluster string) (bool, error)
	BatchCreate(configs []*model.AppEnvConfig) error
}

type appEnvConfigRepository struct {
	db *gorm.DB
}

func NewAppEnvConfigRepository(db *gorm.DB) AppEnvConfigRepository {
	return &appEnvConfigRepository{db: db}
}

func (r *appEnvConfigRepository) Create(config *model.AppEnvConfig) error {
	if err := r.db.Create(config).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建应用环境配置失败", err)
	}
	return nil
}

func (r *appEnvConfigRepository) Update(config *model.AppEnvConfig) error {
	if err := r.db.Save(config).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新应用环境配置失败", err)
	}
	return nil
}

func (r *appEnvConfigRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.AppEnvConfig{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除应用环境配置失败", err)
	}
	return nil
}

func (r *appEnvConfigRepository) FindByID(id int64) (*model.AppEnvConfig, error) {
	var config model.AppEnvConfig
	if err := r.db.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用环境配置失败", err)
	}
	return &config, nil
}

func (r *appEnvConfigRepository) FindByAppID(appID int64) ([]*model.AppEnvConfig, error) {
	var configs []*model.AppEnvConfig
	if err := r.db.Where("app_id = ? AND status = ?", appID, constants.StatusEnabled).
		Order("env, cluster").
		Find(&configs).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用环境配置列表失败", err)
	}
	return configs, nil
}

func (r *appEnvConfigRepository) FindByAppIDAndEnv(appID int64, env string) ([]*model.AppEnvConfig, error) {
	var configs []*model.AppEnvConfig
	if err := r.db.Where("app_id = ? AND env = ? AND status = ?", appID, env, constants.StatusEnabled).
		Order("cluster").
		Find(&configs).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用环境配置失败", err)
	}
	return configs, nil
}

func (r *appEnvConfigRepository) CheckExists(appID int64, env, cluster string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.AppEnvConfig{}).
		Where("app_id = ? AND env = ? AND cluster = ?", appID, env, cluster).
		Count(&count).Error; err != nil {
		return false, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "检查应用环境配置存在性失败", err)
	}
	return count > 0, nil
}

func (r *appEnvConfigRepository) BatchCreate(configs []*model.AppEnvConfig) error {
	if len(configs) == 0 {
		return nil
	}
	if err := r.db.Create(&configs).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "批量创建应用环境配置失败", err)
	}
	return nil
}
