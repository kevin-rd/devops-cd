package dto

import "time"

// CreateApplicationRequest 创建应用请求
type CreateApplicationRequest struct {
	Name        string              `json:"name" binding:"required,max=100"`
	DisplayName *string             `json:"display_name"`
	Description *string             `json:"description"`
	ProjectID   *int64              `json:"project_id"` // 可选：如果不指定则从 Repository 继承
	RepoID      int64               `json:"repo_id" binding:"required"`
	AppType     string              `json:"app_type" binding:"required,oneof=static node java go py"`
	TeamID      *int64              `json:"team_id"`
	EnvClusters map[string][]string `json:"env_clusters,omitempty"` // 环境集群配置，用于初始化 app_env_configs 表
}

// UpdateApplicationRequest 更新应用请求
type UpdateApplicationRequest struct {
	ID          int64               `json:"id" binding:"required"` // 必填：应用ID
	Name        *string             `json:"name" binding:"omitempty,max=100"`
	DisplayName *string             `json:"display_name"`
	Description *string             `json:"description"`
	AppType     *string             `json:"app_type" binding:"omitempty,oneof=static node java go py"`
	TeamID      *int64              `json:"team_id"`
	DeployedTag *string             `json:"deployed_tag"`           // 当前部署的镜像标签
	EnvClusters map[string][]string `json:"env_clusters,omitempty"` // 环境集群配置，用于同步更新 app_env_configs 表
	Status      *int8               `json:"status" binding:"omitempty,oneof=0 1"`
}

// GetApplicationRequest 获取应用详情请求
type GetApplicationRequest struct {
	ID int64 `form:"id" binding:"required"` // 必填：应用ID
}

// DeleteApplicationRequest 删除应用请求
type DeleteApplicationRequest struct {
	ID int64 `json:"id" binding:"required"` // 必填：应用ID
}

// GetApplicationBuildsRequest 获取应用构建历史请求
type GetApplicationBuildsRequest struct {
	ID int64 `form:"id" binding:"required"` // 必填：应用ID
	PageQuery
}

// ApplicationResponse 应用响应
type ApplicationResponse struct {
	ID               int64               `json:"id"`
	Name             string              `json:"name"`
	Namespace        string              `json:"namespace"` // 命名空间（从 Repository 继承）
	Description      *string             `json:"description"`
	ProjectID        int64               `json:"project_id"`             // 关联的项目ID
	ProjectName      *string             `json:"project_name,omitempty"` // 项目名称
	RepoID           int64               `json:"repo_id"`
	RepoName         *string             `json:"repo_name,omitempty"` // Repository的namespace/name
	AppType          string              `json:"app_type"`
	TeamID           *int64              `json:"team_id"`
	TeamName         *string             `json:"team_name,omitempty"` // 团队名称
	DeployedTag      *string             `json:"deployed_tag"`        // 当前部署的镜像标签
	DefaultDependsOn []int64             `json:"default_depends_on"`
	EnvClusters      map[string][]string `json:"env_clusters,omitempty"` // 环境集群配置，从 app_env_configs 表查询得出
	Status           int8                `json:"status"`
	CreatedAt        string              `json:"created_at"`
	UpdatedAt        string              `json:"updated_at"`
}

// ApplicationListQuery 应用列表查询参数
// 所有参数都是可选的，可以任意组合使用
type ApplicationListQuery struct {
	PageQuery         // 分页参数（page, page_size, keyword, status）
	ProjectID *int64  `form:"project_id"`                                                // 可选：按项目ID过滤
	RepoID    *int64  `form:"repo_id"`                                                   // 可选：按代码库ID过滤
	TeamID    *int64  `form:"team_id"`                                                   // 可选：按团队ID过滤
	AppType   *string `form:"app_type" binding:"omitempty,oneof=static node java go py"` // 可选：按应用类型过滤
}

// ApplicationBuildInfo 应用构建信息（简化版）
type ApplicationBuildInfo struct {
	ID            int64   `json:"id"`
	BuildNumber   string  `json:"build_number"`
	Tag           string  `json:"tag"`
	Branch        string  `json:"branch"`
	CommitID      string  `json:"commit_id"`
	CommitMessage *string `json:"commit_message"`
	BuildStatus   string  `json:"build_status"`
	TriggerType   string  `json:"trigger_type"`
	StartedAt     *string `json:"started_at"`
	FinishedAt    *string `json:"finished_at"`
	Duration      *int    `json:"duration"`
	CreatedAt     string  `json:"created_at"`
}

// ApplicationSearchQuery 应用搜索查询参数（包含构建信息）
type ApplicationSearchQuery struct {
	PageQuery          // 分页参数（page, page_size, keyword）
	ProjectID *int64   `form:"project_id"` // 可选：按项目ID过滤
	RepoID    *int64   `form:"repo_id"`    // 可选：按代码库ID过滤
	TeamIDs   []int64  `form:"team_ids"`   // 可选：按团队ID过滤（多选）
	AppTypes  []string `form:"app_types"`  // 可选：按应用类型过滤（多选）
}

// LatestBuildInfo 最新构建信息
type LatestBuildInfo struct {
	BuildID       int64   `json:"build_id"`
	BuildNumber   int     `json:"build_number"`
	ImageTag      string  `json:"image_tag"`
	CommitSHA     string  `json:"commit_sha"`
	CommitMessage *string `json:"commit_message"`
	CommitBranch  string  `json:"commit_branch"`
	BuildStatus   string  `json:"build_status"`
	CreatedAt     string  `json:"created_at"`
}

// ApplicationBuildResponse 应用搜索响应（包含构建信息）
type ApplicationBuildResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Namespace    string  `json:"namespace"` // 命名空间（从 Repository 继承）
	Description  *string `json:"description"`
	ProjectID    int64   `json:"project_id"`             // 关联的项目ID
	ProjectName  *string `json:"project_name,omitempty"` // 项目名称
	RepoID       int64   `json:"repo_id"`
	RepoFullName *string `json:"repo_full_name,omitempty"` // Repository的namespace/name
	TeamID       *int64  `json:"team_id"`
	TeamName     *string `json:"team_name,omitempty"` // 团队名称

	AppType     string  `json:"app_type"`
	DeployedTag *string `json:"deployed_tag"` // 当前部署的镜像标签
	Status      int8    `json:"status"`

	BuildID       int64      `json:"build_id"`
	BuildNumber   int64      `json:"build_number"`
	BuildTime     *time.Time `json:"build_time"`
	ImageTag      string     `json:"image_tag"`
	CommitSHA     string     `json:"commit_sha"`
	CommitMessage *string    `json:"commit_message"`
	CommitBranch  string     `json:"commit_branch"`
	BuildStatus   string     `json:"build_status"`
}

// UpdateAppDependenciesRequest 更新应用默认依赖请求
type UpdateAppDependenciesRequest struct {
	Dependencies []int64 `json:"dependencies"`
}

// ApplicationDependenciesResponse 应用默认依赖响应
type ApplicationDependenciesResponse struct {
	AppID        int64   `json:"app_id"`
	AppName      string  `json:"app_name"`
	Dependencies []int64 `json:"dependencies"`
	UpdatedAt    string  `json:"updated_at"`
}
