package dto

import "encoding/json"

// ProjectEnvConfigRequest 项目环境配置请求（用于批量更新）
type ProjectEnvConfigRequest struct {
	AllowClusters   []string        `json:"allow_clusters" binding:"required"`
	DefaultClusters []string        `json:"default_clusters"`
	SchemaVersion   *int            `json:"schema_version"`
	ArtifactsJSON   json.RawMessage `json:"artifacts_json"` // v1 统一配置（可选，优先级高于旧字段）
}

// UpdateProjectEnvConfigsRequest 批量更新项目环境配置请求
type UpdateProjectEnvConfigsRequest struct {
	Configs map[string]*ProjectEnvConfigRequest `json:"configs" binding:"required"` // key: env (pre/prod)
}

// ProjectEnvConfigResponse 项目环境配置响应
type ProjectEnvConfigResponse struct {
	ID              int64           `json:"id"`
	ProjectID       int64           `json:"project_id"`
	Env             string          `json:"env"`
	AllowClusters   []string        `json:"allow_clusters"`
	DefaultClusters []string        `json:"default_clusters"`
	SchemaVersion   int             `json:"schema_version"`
	ArtifactsJSON   json.RawMessage `json:"artifacts_json,omitempty"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}
