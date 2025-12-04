package dto

// ProjectEnvConfigRequest 项目环境配置请求（用于批量更新）
type ProjectEnvConfigRequest struct {
	AllowClusters          []string `json:"allow_clusters" binding:"required"`
	DefaultClusters        []string `json:"default_clusters"`
	Namespace              *string  `json:"namespace" binding:"max=63"`
	DeploymentNameTemplate *string  `json:"deployment_name_template" binding:"max=255"`
	ChartRepoURL           *string  `json:"chart_repo_url" binding:"max=255"`
}

// UpdateProjectEnvConfigsRequest 批量更新项目环境配置请求
type UpdateProjectEnvConfigsRequest struct {
	Configs map[string]*ProjectEnvConfigRequest `json:"configs" binding:"required"` // key: env (pre/prod)
}

// ProjectEnvConfigResponse 项目环境配置响应
type ProjectEnvConfigResponse struct {
	ID                     int64    `json:"id"`
	ProjectID              int64    `json:"project_id"`
	Env                    string   `json:"env"`
	AllowClusters          []string `json:"allow_clusters"`
	DefaultClusters        []string `json:"default_clusters"`
	Namespace              string   `json:"namespace"`
	DeploymentNameTemplate string   `json:"deployment_name_template"`
	ChartRepoURL           string   `json:"chart_repo_url"`
	CreatedAt              string   `json:"created_at"`
	UpdatedAt              string   `json:"updated_at"`
}
