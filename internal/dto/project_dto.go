package dto

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name              string  `json:"name" binding:"required,max=100"`
	Description       *string `json:"description"`
	OwnerName         *string `json:"owner_name" binding:"omitempty,max=100"`
	CreateDefaultTeam *bool   `json:"create_default_team" binding:"omitempty"`
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	ID          int64   `json:"id" binding:"required"`
	Name        *string `json:"name" binding:"omitempty,max=100"`
	Description *string `json:"description"`
	OwnerName   *string `json:"owner_name" binding:"omitempty,max=100"`
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
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	OwnerName   *string         `json:"owner_name"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Teams       []*TeamResponse `json:"teams,omitempty"`
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
