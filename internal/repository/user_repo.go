package repository

import (
	pkgErrors "devops-cd/pkg/responses"
	"errors"
	"gorm.io/gorm"

	"devops-cd/internal/model"
	"devops-cd/internal/pkg/database"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		db: database.GetDB(),
	}
}

func (r *UserRepository) Create(user *model.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建用户失败", err)
	}
	return nil
}

func (r *UserRepository) FindByUsername(username, authProvider string) (*model.User, error) {
	var user model.User
	err := r.queryByUsername(username, authProvider).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询用户失败", err)
	}
	return &user, nil
}

func (r *UserRepository) FindWithTeams(username, authProvider string) (*model.User, error) {
	var user model.User
	if err := r.queryByUsername(username, authProvider).Preload("TeamMembers").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询用户失败", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询用户失败", err)
	}
	return &user, nil
}

func (r *UserRepository) Search(keyword string, page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	query := r.db.Model(&model.User{}).Where("deleted_at IS NULL").Where("status = 1")

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计用户失败", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询用户失败", err)
	}

	return users, total, nil
}

func (r *UserRepository) Update(user *model.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新用户失败", err)
	}
	return nil
}

func (r *UserRepository) UpdateLastLogin(id int64) error {
	if err := r.db.Model(&model.User{}).Where("id = ?", id).Update("last_login_at", gorm.Expr("NOW()")).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新登录时间失败", err)
	}
	return nil
}

func (r *UserRepository) queryByUsername(username, authProvider string) *gorm.DB {
	db := r.db.Where("username = ? AND deleted_at IS NULL", username)
	if authProvider != "" {
		db = db.Where("auth_provider = ?", authProvider)
	}
	return db
}
