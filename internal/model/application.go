package model

import (
	"time"
)

const ApplicationTableName = "applications"

// Application 应用模型
type Application struct {
	BaseStatus

	RepoID    int64  `gorm:"column:repo_id;not null;index" json:"repo_id"`
	ProjectID int64  `gorm:"column:project_id;not null;uniqueIndex:uk_project_app_name;index" json:"project_id"`
	TeamID    *int64 `gorm:"index" json:"team_id"`

	Name             string    `gorm:"size:100;not null;uniqueIndex:uk_project_app_name" json:"name"`
	Description      *string   `gorm:"type:text" json:"description"`
	AppType          string    `gorm:"size:50;not null;index" json:"app_type"`
	DeployedTag      *string   `gorm:"column:deployed_tag;size:100" json:"deployed_tag"`              // 当前部署的镜像标签
	DefaultDependsOn Int64List `gorm:"column:default_depends_on;type:json" json:"default_depends_on"` // DefaultDependsOn 配置级依赖（JSON 数组，记录应用 ID）

	// Relations
	Repository *Repository    `gorm:"foreignKey:RepoID" json:"repository,omitempty"`
	Project    *Project       `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Team       *Team          `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	EnvConfigs []AppEnvConfig `gorm:"foreignKey:AppID" json:"env_configs,omitempty"`
}

func (Application) TableName() string {
	return ApplicationTableName
}

// ApplicationWithBuild 应用及其最新构建信息（用于搜索和列表展示）
type ApplicationWithBuild struct {
	Application // 嵌入 Application，继承所有字段

	// 最新构建信息（用于排序、过滤和显示）
	LatestBuildID        *int64     `gorm:"column:latest_build_id" json:"latest_build_id,omitempty"`
	LatestBuildNumber    *int64     `gorm:"column:latest_build_number" json:"latest_build_number,omitempty"`
	LatestImageTag       *string    `gorm:"column:latest_image_tag" json:"latest_image_tag,omitempty"`
	LatestCommitSHA      *string    `gorm:"column:latest_commit_sha" json:"latest_commit_sha,omitempty"`
	LatestCommitMessage  *string    `gorm:"column:latest_commit_message" json:"latest_commit_message,omitempty"`
	LatestCommitBranch   *string    `gorm:"column:latest_commit_branch" json:"latest_commit_branch,omitempty"`
	LatestBuildStatus    *string    `gorm:"column:latest_build_status" json:"latest_build_status,omitempty"`
	LatestBuildCreatedAt *time.Time `gorm:"column:latest_build_created_at" json:"latest_build_created_at,omitempty"`
}

// TableName 指定表名（仍然查询 applications 表）
func (ApplicationWithBuild) TableName() string {
	return ApplicationTableName
}
