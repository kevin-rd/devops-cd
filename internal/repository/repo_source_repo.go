package repository

import (
	pkgErrors "devops-cd/pkg/responses"
	"time"

	"devops-cd/internal/model"
	"gorm.io/gorm"
)

type RepoSyncSourceRepository interface {
	Create(source *model.RepoSource) error
	Update(source *model.RepoSource) error
	Delete(id int64) error
	GetByID(id int64) (*model.RepoSource, error)
	List(page, pageSize int, keyword, platform, baseURL, namespace string, enabled *bool) ([]*model.RepoSource, int64, error)
	ListEnabled() ([]*model.RepoSource, error)
	UpdateSyncResult(id int64, status string, message *string) error
}

type repoSyncSourceRepository struct {
	db *gorm.DB
}

func NewRepoSyncSourceRepository(db *gorm.DB) RepoSyncSourceRepository {
	return &repoSyncSourceRepository{db: db}
}

func (r *repoSyncSourceRepository) Create(source *model.RepoSource) error {
	if err := r.db.Create(source).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建仓库源失败", err)
	}
	return nil
}

func (r *repoSyncSourceRepository) Update(source *model.RepoSource) error {
	if err := r.db.Save(source).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新仓库源失败", err)
	}
	return nil
}

func (r *repoSyncSourceRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.RepoSource{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除仓库源失败", err)
	}
	return nil
}

func (r *repoSyncSourceRepository) GetByID(id int64) (*model.RepoSource, error) {
	var source model.RepoSource
	if err := r.db.Preload("DefaultProject").Preload("DefaultTeam").First(&source, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询仓库源失败", err)
	}
	return &source, nil
}

func (r *repoSyncSourceRepository) List(page, pageSize int, keyword, platform, baseURL, namespace string, enabled *bool) ([]*model.RepoSource, int64, error) {
	var (
		sources []*model.RepoSource
		total   int64
	)

	query := r.db.Model(&model.RepoSource{})

	if keyword != "" {
		query = query.Where("(base_url LIKE ? OR namespace LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if baseURL != "" {
		query = query.Where("base_url LIKE ?", "%"+baseURL+"%")
	}
	if namespace != "" {
		query = query.Where("namespace LIKE ?", "%"+namespace+"%")
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计仓库源失败", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("DefaultProject").Preload("DefaultTeam").
		Offset(offset).Limit(pageSize).Order("updated_at DESC").Find(&sources).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询仓库源列表失败", err)
	}

	return sources, total, nil
}

func (r *repoSyncSourceRepository) ListEnabled() ([]*model.RepoSource, error) {
	var sources []*model.RepoSource
	if err := r.db.Preload("DefaultProject").Preload("DefaultTeam").
		Where("enabled = ?", true).Find(&sources).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询启用的仓库源失败", err)
	}
	return sources, nil
}

func (r *repoSyncSourceRepository) UpdateSyncResult(id int64, status string, message *string) error {
	updates := map[string]interface{}{
		"last_synced_at": time.Now(),
		"last_status":    status,
	}
	if message != nil {
		updates["last_message"] = *message
	} else {
		updates["last_message"] = nil
	}

	if err := r.db.Model(&model.RepoSource{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新同步结果失败", err)
	}
	return nil
}
