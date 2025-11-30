package model

import "time"

const BuildTableName = "builds"

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

	// 构建时间
	BuildCreated  time.Time `gorm:"not null" json:"build_created"`
	BuildStarted  time.Time `gorm:"not null" json:"build_started"`
	BuildFinished time.Time `gorm:"not null" json:"build_finished"`
	BuildDuration int       `json:"build_duration"` // 构建耗时（秒）

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
	return BuildTableName
}
