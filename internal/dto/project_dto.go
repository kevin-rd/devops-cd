package dto

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name               string               `json:"name" binding:"required,max=100"`
	Description        *string              `json:"description"`
	OwnerName          *string              `json:"owner_name" binding:"omitempty,max=100"`
	CreateDefaultTeam  *bool                `json:"create_default_team" binding:"omitempty"`
	AllowedEnvClusters *map[string][]string `json:"allowed_env_clusters"` // 允许的环境集群配置: {"pre": ["cluster-a"], "prod": ["cluster-b"]}
	DefaultEnvClusters *map[string][]string `json:"default_env_clusters"` // 项目默认环境集群配置(必须是 allowed_env_clusters 的子集)
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	ID                 int64                `json:"id" binding:"required"`
	Name               *string              `json:"name" binding:"omitempty,max=100"`
	Description        *string              `json:"description"`
	OwnerName          *string              `json:"owner_name" binding:"omitempty,max=100"`
	AllowedEnvClusters *map[string][]string `json:"allowed_env_clusters"` // 允许的环境集群配置
	DefaultEnvClusters *map[string][]string `json:"default_env_clusters"` // 项目默认环境集群配置(必须是 allowed_env_clusters 的子集)
}

// DeleteProjectRequest 删除项目请求
type DeleteProjectRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// GetProjectRequest 获取项目详情请求
type GetProjectRequest struct {
	ID int64 `form:"id" binding:"required"`
}

// ProjectResponse 项目响应
type ProjectResponse struct {
	ID                 int64                `json:"id"`
	Name               string               `json:"name"`
	Description        *string              `json:"description"`
	OwnerName          *string              `json:"owner_name"`
	AllowedEnvClusters *map[string][]string `json:"allowed_env_clusters"` // 允许的环境集群配置
	DefaultEnvClusters *map[string][]string `json:"default_env_clusters"` // 项目默认环境集群配置(必须是 allowed_env_clusters 的子集)
	CreatedAt          string               `json:"created_at"`
	UpdatedAt          string               `json:"updated_at"`
	Teams              []*TeamResponse      `json:"teams,omitempty"`
}

// ProjectListQuery 项目列表查询参数
type ProjectListQuery struct {
	PageQuery
	WithTeams bool `form:"with_teams"`
}

// ProjectSimpleResponse 项目简单响应（用于下拉选择）
type ProjectSimpleResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GetProjectAvailableEnvClustersRequest 获取项目可用环境集群请求
type GetProjectAvailableEnvClustersRequest struct {
	ProjectID int64  `form:"project_id" binding:"required"`
	Env       string `form:"env"` // 可选:指定环境,返回该环境下可用集群列表
}

// ProjectAvailableEnvClustersResponse 项目可用环境集群响应
type ProjectAvailableEnvClustersResponse struct {
	AllowedEnvClusters map[string][]string `json:"allowed_env_clusters"` // 完整配置: {"pre": ["cluster-a"], "prod": ["cluster-b"]}
	AvailableClusters  []string            `json:"available_clusters"`   // 如果指定了env,返回该环境下可用集群列表
}
