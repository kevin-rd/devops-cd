package repository

import "gorm.io/gorm"

type QueryOption func(*gorm.DB) *gorm.DB

func WithPreload(association string, conds ...interface{}) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Preload(association, conds...)
	}
}
