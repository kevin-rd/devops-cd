package dto

// CreateRepositoryRequest 创建代码库请求
type CreateRepositoryRequest struct {
	Project     string  `json:"project" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description"`
	GitURL      string  `json:"git_url" binding:"required,url"`
	GitType     string  `json:"git_type" binding:"required,oneof=gitea gitlab github"`
	Language    *string `json:"language"`
	TeamID      *int64  `json:"team_id"`
}

// UpdateRepositoryRequest 更新代码库请求
type UpdateRepositoryRequest struct {
	ID          int64   `json:"id" binding:"required"` // 必填：要更新的代码库ID
	Project     *string `json:"project" binding:"omitempty,max=100"`
	Name        *string `json:"name" binding:"omitempty,max=100"`
	Description *string `json:"description"`
	GitURL      *string `json:"git_url" binding:"omitempty,url"`
	GitType     *string `json:"git_type" binding:"omitempty,oneof=gitea gitlab github"`
	Language    *string `json:"language"`
	TeamID      *int64  `json:"team_id"`
	Status      *int8   `json:"status" binding:"omitempty,oneof=0 1"`
}

// DeleteRepositoryRequest 删除代码库请求（软删除）
type DeleteRepositoryRequest struct {
	ID int64 `json:"id" binding:"required"` // 必填：要删除的代码库ID
}

// GetRepositoryRequest 获取代码库详情请求
type GetRepositoryRequest struct {
	ID int64 `form:"id" binding:"required"` // 必填：代码库ID
}

// RepositoryResponse 代码库响应
type RepositoryResponse struct {
	ID           int64                  `json:"id"`
	Project      string                 `json:"project"`
	Name         string                 `json:"name"`
	FullName     string                 `json:"full_name"` // project/name
	Description  *string                `json:"description"`
	GitURL       string                 `json:"git_url"`
	GitType      string                 `json:"git_type"`
	Language     *string                `json:"language"`
	TeamID       *int64                 `json:"team_id"`
	TeamName     *string                `json:"team_name,omitempty"`
	Status       int8                   `json:"status"`
	Applications []*ApplicationResponse `json:"applications,omitempty"` // 关联的应用列表
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// RepositoryListQuery 代码库列表查询参数
// 所有参数都是可选的，可以任意组合使用
type RepositoryListQuery struct {
	PageQuery                // 分页参数（page, page_size, keyword, status）
	Project          *string `form:"project"`                                                // 可选：按项目名称过滤
	TeamID           *int64  `form:"team_id"`                                                // 可选：按团队ID过滤
	GitType          *string `form:"git_type" binding:"omitempty,oneof=gitea gitlab github"` // 可选：按Git类型过滤
	WithApplications *bool   `form:"with_applications"`                                      // 可选：是否包含应用列表，默认false
}
