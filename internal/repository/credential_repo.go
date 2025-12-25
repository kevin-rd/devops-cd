package repository

import (
	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/responses"
	"gorm.io/gorm"
)

type CredentialRepository struct {
	db *gorm.DB
}

func NewCredentialRepository(db *gorm.DB) *CredentialRepository {
	return &CredentialRepository{db: db}
}

func (r *CredentialRepository) Create(c *model.Credential) error {
	if err := r.db.Create(c).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建凭据失败", err)
	}
	return nil
}

func (r *CredentialRepository) GetByID(id int64) (*model.Credential, error) {
	var c model.Credential
	if err := r.db.First(&c, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询凭据失败", err)
	}
	return &c, nil
}

func (r *CredentialRepository) Update(c *model.Credential) error {
	if err := r.db.Save(c).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新凭据失败", err)
	}
	return nil
}

func (r *CredentialRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.Credential{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除凭据失败", err)
	}
	return nil
}

func (r *CredentialRepository) List(scope string, projectID *int64) ([]*model.Credential, error) {
	var list []*model.Credential
	q := r.db.Model(&model.Credential{})
	if scope != "" {
		q = q.Where("scope = ?", scope)
	}
	if projectID != nil {
		q = q.Where("project_id = ?", *projectID)
	}
	if err := q.Order("id DESC").Find(&list).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询凭据列表失败", err)
	}
	return list, nil
}
