package repository

import (
	"devops-cd/internal/model"
	"gorm.io/gorm"
)

type ReleaseAppRepository struct {
	db *gorm.DB
}

func NewReleaseAppRepository(db *gorm.DB) *ReleaseAppRepository {
	return &ReleaseAppRepository{
		db: db,
	}
}

func (r *ReleaseAppRepository) Create(app *model.ReleaseApp) error {
	return r.db.Create(app).Error
}

func (r *ReleaseAppRepository) DeleteByAppIDs(tx *gorm.DB, batchID int64, ids []int64) error {
	return tx.Where("batch_id = ? AND app_id IN ?", batchID, ids).Delete(&model.ReleaseApp{}).Error
}
