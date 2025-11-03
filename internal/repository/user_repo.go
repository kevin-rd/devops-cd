package repository

import (
	"gorm.io/gorm"

	"devops-cd/internal/model"
	"devops-cd/internal/pkg/database"
	pkgErrors "devops-cd/pkg/errors"
)

type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByID(id int64) (*model.User, error)
	Update(user *model.User) error
	UpdateLastLogin(id int64) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository() UserRepository {
	return &userRepository{
		db: database.GetDB(),
	}
}

func (r *userRepository) Create(user *model.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建用户失败", err)
	}
	return nil
}

func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询用户失败", err)
	}
	return &user, nil
}

func (r *userRepository) FindByID(id int64) (*model.User, error) {
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

func (r *userRepository) Update(user *model.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新用户失败", err)
	}
	return nil
}

func (r *userRepository) UpdateLastLogin(id int64) error {
	if err := r.db.Model(&model.User{}).Where("id = ?", id).Update("last_login_at", gorm.Expr("NOW()")).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新登录时间失败", err)
	}
	return nil
}
