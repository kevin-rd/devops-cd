package model

import (
	"gorm.io/gorm"
	"time"
)

type BaseModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// BaseModelWithSoftDelete 基础模型
type BaseModelWithSoftDelete struct {
	BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BaseStatus 带状态的基础模型
type BaseStatus struct {
	BaseModelWithSoftDelete
	Status int8 `gorm:"not null;default:1;index" json:"status"` // 1:启用 0:禁用
}
