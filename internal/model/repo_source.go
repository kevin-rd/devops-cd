package model

import (
	"time"

	"gorm.io/datatypes"
)

const RepoSourceTableName = "repo_sources"

// RepoSource 存储 Git 仓库同步源配置（按 namespace 粒度）
type RepoSource struct {
	BaseModelWithSoftDelete

	Platform         string            `gorm:"size:20;not null" json:"platform"`
	BaseURL          string            `gorm:"size:255;not null;index:idx_repo_source_base_namespace,priority:1" json:"base_url"`
	Namespace        string            `gorm:"size:255;not null;index:idx_repo_source_base_namespace,priority:2" json:"namespace"`
	AuthTokenEnc     string            `gorm:"type:text;not null" json:"-"`
	Enabled          bool              `gorm:"not null;default:true;index" json:"enabled"`
	DefaultProjectID *int64            `gorm:"index" json:"default_project_id"` // 默认项目ID
	DefaultTeamID    *int64            `gorm:"index" json:"default_team_id"`    // 默认团队ID
	LastSyncedAt     *time.Time        `json:"last_synced_at"`
	LastStatus       *string           `gorm:"size:20" json:"last_status"`
	LastMessage      *string           `gorm:"type:text" json:"last_message"`
	Ext              datatypes.JSONMap `gorm:"type:json" json:"ext,omitempty"`
	CreatedBy        *string           `gorm:"size:50" json:"created_by"`
	UpdatedBy        *string           `gorm:"size:50" json:"updated_by"`

	// 关联关系（仅用于加载，不存储）
	DefaultProject *Project `gorm:"foreignKey:DefaultProjectID" json:"default_project,omitempty"`
	DefaultTeam    *Team    `gorm:"foreignKey:DefaultTeamID" json:"default_team,omitempty"`
}

func (RepoSource) TableName() string {
	return RepoSourceTableName
}
