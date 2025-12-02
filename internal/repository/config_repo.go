package repository

import (
	"database/sql"
	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/errors"
	"errors"
	"gorm.io/gorm"
)

type ConfigRepository struct {
	db *gorm.DB
}

func NewConfigRepository(db *gorm.DB) *ConfigRepository {
	return &ConfigRepository{
		db: db,
	}
}

// GetConfig 获取配置值，优先返回项目级配置，如果不存在则返回全局配置
func (r *ConfigRepository) GetConfig(projectID int64, key string) (string, error) {
	var configItem model.ConfigItem

	// 优先查询项目级配置
	err := r.db.Where("scope = ? AND project_id = ?", model.ScopeProject, projectID).Where("config_key = ?", key).First(&configItem).Error
	if err == nil {
		// 找到项目级配置，直接返回
		return r.resolveConfigValue(configItem)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目级配置失败", err)
	}

	// 项目级配置不存在，fallback 到全局配置
	err = r.db.Where("scope = ? AND config_key = ?", model.ScopeGlobal, key).First(&configItem).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", pkgErrors.ErrRecordNotFound
		}
		return "", pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询全局配置失败", err)
	}

	// 找到全局配置，返回
	return r.resolveConfigValue(configItem)
}

// resolveConfigValue 解析配置值，如果是 secret 类型则解密
func (r *ConfigRepository) resolveConfigValue(configItem model.ConfigItem) (string, error) {
	if configItem.ValueType == model.TypeSecret {
		return resolveSecret(configItem.Value)
	}
	return configItem.Value, nil
}

// SetConfig 设置配置值，如果已存在则更新，不存在则创建
func (r *ConfigRepository) SetConfig(scope model.Scope, projectID *int64, key string, value string, valueType model.ValueType, updatedBy int64) error {
	var configItem model.ConfigItem

	// 构建查询条件
	query := r.db.Where("scope = ? AND key = ?", scope, key)
	if scope == model.ScopeProject {
		if projectID == nil {
			return pkgErrors.New(pkgErrors.CodeBadRequest, "项目级配置必须指定 project_id")
		}
		query = query.Where("project_id = ?", *projectID)
	} else {
		// 全局配置的 project_id 应该为 NULL
		query = query.Where("project_id IS NULL")
	}

	// 查询是否已存在
	err := query.First(&configItem).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询配置失败", err)
	}

	// 设置配置项的值
	configItem.Scope = scope
	if projectID != nil {
		configItem.ProjectID = sql.NullInt64{Int64: *projectID, Valid: true}
	} else {
		configItem.ProjectID = sql.NullInt64{Valid: false}
	}
	configItem.Key = key
	configItem.Value = value
	configItem.ValueType = valueType

	// 如果已存在，更新；否则创建
	if err == gorm.ErrRecordNotFound {
		if err := r.db.Create(&configItem).Error; err != nil {
			return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建配置失败", err)
		}
	} else {
		if err := r.db.Save(&configItem).Error; err != nil {
			return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新配置失败", err)
		}
	}

	return nil
}

// 假设你有 Vault 或 AES secret 解密函数
func resolveSecret(ref string) (string, error) {
	// 这里示例直接返回原值，生产中替换成 SecretStore 读取
	return ref, nil
}
