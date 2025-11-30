package model

const ProjectTableName = "projects"

// Project 项目模型
type Project struct {
	BaseModelWithSoftDelete
	Name               string  `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description        *string `gorm:"type:text" json:"description"`
	OwnerName          *string `gorm:"size:100" json:"owner_name"`
	AllowedEnvClusters *string `gorm:"type:json;column:allowed_env_clusters" json:"allowed_env_clusters"` // 允许的环境集群配置
	DefaultEnvClusters *string `gorm:"type:json;column:default_env_clusters" json:"default_env_clusters"` // 项目默认环境集群配置(必须是 AllowedEnvClusters 的子集)
}

func (Project) TableName() string {
	return ProjectTableName
}
