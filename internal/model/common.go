package model

import (
	"gorm.io/gorm"
	"time"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BaseStatus 带状态的基础模型
type BaseStatus struct {
	BaseModel
	Status int8 `gorm:"not null;default:1;index" json:"status"` // 1:启用 0:禁用
}
