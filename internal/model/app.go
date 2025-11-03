package model

import "time"

// Repository 代码库模型
type Repository struct {
	BaseStatus
	Project     string  `gorm:"size:100;not null;index:idx_project_name" json:"project"`
	Name        string  `gorm:"size:100;not null;uniqueIndex:idx_project_name" json:"name"`
	Description *string `gorm:"type:text" json:"description"`
	GitURL      string  `gorm:"size:500;not null" json:"git_url"`
	GitType     string  `gorm:"size:20;not null;index" json:"git_type"`
	Language    *string `gorm:"size:50" json:"language"`
	TeamID      *int64  `gorm:"index" json:"team_id"`

	// Relations
	Team *Team `gorm:"foreignKey:TeamID" json:"team,omitempty"`
}

func (Repository) TableName() string {
	return "repositories"
}

// Application 应用模型
type Application struct {
	BaseStatus
	Name        string  `gorm:"size:100;not null;uniqueIndex:uk_project_app_name" json:"name"`
	Project     string  `gorm:"size:100;not null;uniqueIndex:uk_project_app_name;index" json:"project"`
	DisplayName *string `gorm:"size:100" json:"display_name"`
	Description *string `gorm:"type:text" json:"description"`
	RepoID      int64   `gorm:"column:repo_id;not null;index" json:"repo_id"`
	AppType     string  `gorm:"size:50;not null;index" json:"app_type"`
	TeamID      *int64  `gorm:"index" json:"team_id"`
	DeployedTag *string `gorm:"column:deployed_tag;size:100" json:"deployed_tag"` // 当前部署的镜像标签

	// Relations
	Repository *Repository `gorm:"foreignKey:RepoID" json:"repository,omitempty"`
	Team       *Team       `gorm:"foreignKey:TeamID" json:"team,omitempty"`
}

func (Application) TableName() string {
	return "applications"
}

// Build 构建记录
type Build struct {
	ID     int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	RepoID int64 `gorm:"column:repo_id;not null;index:idx_repo_build" json:"repo_id"`
	AppID  int64 `gorm:"column:app_id;not null;uniqueIndex:uk_app_build_number" json:"app_id"`

	// 构建信息
	BuildNumber int    `gorm:"not null;uniqueIndex:uk_app_build_number;index:idx_repo_build" json:"build_number"`
	BuildStatus string `gorm:"size:20;not null;index" json:"build_status"` // success/failure/error/killed
	BuildEvent  string `gorm:"size:20;not null;index" json:"build_event"`  // push/tag/pull_request/promote/rollback
	BuildLink   string `gorm:"size:255" json:"build_link"`

	// Git 提交信息
	CommitSHA     string `gorm:"column:commit_sha;size:64;not null;index" json:"commit_sha"`
	CommitRef     string `gorm:"size:255" json:"commit_ref"`
	CommitBranch  string `gorm:"size:100" json:"commit_branch"`
	CommitMessage string `gorm:"type:text" json:"commit_message"`
	CommitLink    string `gorm:"size:255" json:"commit_link"`
	CommitAuthor  string `gorm:"size:100" json:"commit_author"`

	// 构建时间（Unix 时间戳，秒）
	BuildCreated  int64 `gorm:"not null" json:"build_created"`
	BuildStarted  int64 `gorm:"not null" json:"build_started"`
	BuildFinished int64 `gorm:"not null" json:"build_finished"`
	BuildDuration int   `json:"build_duration"` // 构建耗时（秒）

	// 镜像信息
	ImageTag        string `gorm:"size:100;not null;index" json:"image_tag"`
	ImageURL        string `gorm:"column:image_url;size:500" json:"image_url"`
	AppBuildSuccess bool   `gorm:"not null;default:true" json:"app_build_success"`

	// 环境信息
	Environment string `gorm:"size:50" json:"environment"`

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	Repository  *Repository  `gorm:"foreignKey:RepoID" json:"repository,omitempty"`
	Application *Application `gorm:"foreignKey:AppID" json:"application,omitempty"`
}

// TableName 指定表名
func (Build) TableName() string {
	return "builds"
}

// ApplicationWithBuild 应用及其最新构建信息（用于搜索和列表展示）
type ApplicationWithBuild struct {
	Application // 嵌入 Application，继承所有字段

	// 最新构建信息（用于排序、过滤和显示）
	LatestBuildID        *int64     `gorm:"column:latest_build_id" json:"latest_build_id,omitempty"`
	LatestBuildNumber    *int       `gorm:"column:latest_build_number" json:"latest_build_number,omitempty"`
	LatestImageTag       *string    `gorm:"column:latest_image_tag" json:"latest_image_tag,omitempty"`
	LatestCommitSHA      *string    `gorm:"column:latest_commit_sha" json:"latest_commit_sha,omitempty"`
	LatestCommitMessage  *string    `gorm:"column:latest_commit_message" json:"latest_commit_message,omitempty"`
	LatestCommitBranch   *string    `gorm:"column:latest_commit_branch" json:"latest_commit_branch,omitempty"`
	LatestBuildStatus    *string    `gorm:"column:latest_build_status" json:"latest_build_status,omitempty"`
	LatestBuildCreatedAt *time.Time `gorm:"column:latest_build_created_at" json:"latest_build_created_at,omitempty"`
}

// TableName 指定表名（仍然查询 applications 表）
func (ApplicationWithBuild) TableName() string {
	return "applications"
}
