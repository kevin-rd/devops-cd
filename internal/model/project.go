package model

const ProjectTableName = "projects"

// Project 项目模型
type Project struct {
	BaseModelWithSoftDelete
	Name        string  `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description *string `gorm:"type:text" json:"description"`
	OwnerName   *string `gorm:"size:100" json:"owner_name"`
}

func (Project) TableName() string {
	return ProjectTableName
}
