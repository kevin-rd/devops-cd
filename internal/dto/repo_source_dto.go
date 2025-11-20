package dto

import "time"

type RepoSyncSourceResponse struct {
	ID                 int64      `json:"id"`
	Platform           string     `json:"platform"`
	BaseURL            string     `json:"base_url"`
	Namespace          string     `json:"namespace"`
	Enabled            bool       `json:"enabled"`
	DefaultProjectID   *int64     `json:"default_project_id"`
	DefaultProjectName *string    `json:"default_project_name,omitempty"`
	DefaultTeamID      *int64     `json:"default_team_id"`
	DefaultTeamName    *string    `json:"default_team_name,omitempty"`
	LastSyncedAt       *time.Time `json:"last_synced_at"`
	LastStatus         *string    `json:"last_status"`
	LastMessage        *string    `json:"last_message"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	HasToken           bool       `json:"has_token"`
}

type RepoSyncSourceListQuery struct {
	PageQuery
	Platform  string `form:"platform"`
	BaseURL   string `form:"base_url"`
	Namespace string `form:"namespace"`
	Enabled   *bool  `form:"enabled"`
}

type CreateRepoSyncSourceRequest struct {
	Platform         string  `json:"platform" binding:"required,oneof=gitea gitlab github"`
	BaseURL          string  `json:"base_url" binding:"required,url"`
	Namespace        string  `json:"namespace" binding:"required"`
	Token            string  `json:"token" binding:"required"`
	Enabled          *bool   `json:"enabled"`
	DefaultProjectID *int64  `json:"default_project_id"` // 默认项目ID
	DefaultTeamID    *int64  `json:"default_team_id"`    // 默认团队ID
	CreatedBy        *string `json:"created_by"`
}

type UpdateRepoSyncSourceRequest struct {
	ID               int64   `json:"id" binding:"required"`
	Platform         string  `json:"platform" binding:"required,oneof=gitea gitlab github"`
	BaseURL          string  `json:"base_url" binding:"required,url"`
	Namespace        string  `json:"namespace" binding:"required"`
	Token            *string `json:"token"`
	Enabled          *bool   `json:"enabled"`
	DefaultProjectID *int64  `json:"default_project_id"` // 默认项目ID
	DefaultTeamID    *int64  `json:"default_team_id"`    // 默认团队ID
	UpdatedBy        *string `json:"updated_by"`
}
