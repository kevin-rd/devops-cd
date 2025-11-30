package model

const RepositoryTableName = "repositories"

// Repository 代码库模型
type Repository struct {
	BaseStatus
	Namespace   string  `gorm:"column:namespace;size:100;not null;index:idx_namespace_name" json:"namespace"`
	Name        string  `gorm:"size:100;not null;uniqueIndex:idx_namespace_name" json:"name"`
	Description *string `gorm:"type:text" json:"description"`
	GitURL      string  `gorm:"size:500;not null" json:"git_url"`
	GitType     string  `gorm:"size:20;not null;index" json:"git_type"`
	Language    *string `gorm:"size:50" json:"language"`
	ProjectID   *int64  `gorm:"column:project_id;index" json:"project_id"`
	TeamID      *int64  `gorm:"column:team_id" json:"team_id"`

	// Relations
	Team    *Team    `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

func (Repository) TableName() string {
	return RepositoryTableName
}
