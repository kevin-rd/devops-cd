package model

const ProjectTableName = "projects"
const ProjectEnvConfigTableName = "project_env_configs"

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

// ProjectEnvConfig 项目环境配置模型
type ProjectEnvConfig struct {
	BaseModel

	ProjectID              int64   `gorm:"column:project_id;not null;uniqueIndex:uniq_project_env" json:"project_id"`
	Env                    string  `gorm:"size:32;not null;uniqueIndex:uniq_project_env" json:"env"`
	AllowClusters          string  `gorm:"type:json;column:allow_clusters;not null" json:"allow_clusters"`     // JSON格式的允许集群列表
	DefaultClusters        string  `gorm:"type:json;column:default_clusters;not null" json:"default_clusters"` // JSON格式的默认集群列表
	Namespace              string  `gorm:"size:63;not null;default:''" json:"namespace"`                       // kubernetes命名空间
	DeploymentNameTemplate string  `gorm:"size:255;not null;default:''" json:"deployment_name_template"`       // 部署名称模板
	ChartRepoURL           string  `gorm:"size:255;not null;default:''" json:"chart_repo_url"`                 // Chart仓库URL
	ValuesRepoURL          *string `gorm:"size:255" json:"values_repo_url"`                                    // Values仓库URL
	ValuesPathTemplate     *string `gorm:"size:255" json:"values_path_template"`                               // Values仓库路径模板
}

func (ProjectEnvConfig) TableName() string {
	return ProjectEnvConfigTableName
}
