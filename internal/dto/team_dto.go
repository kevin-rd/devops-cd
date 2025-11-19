package dto

// CreateTeamRequest 创建团队请求
type CreateTeamRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	ProjectID   int64   `json:"project_id" binding:"required"`
	Description *string `json:"description"`
	LeaderName  *string `json:"leader_name" binding:"omitempty,max=100"`
}

// UpdateTeamRequest 更新团队请求
type UpdateTeamRequest struct {
	ID          int64   `json:"id" binding:"required"`
	Name        *string `json:"name" binding:"omitempty,max=100"`
	ProjectID   *int64  `json:"project_id"`
	Description *string `json:"description"`
	LeaderName  *string `json:"leader_name" binding:"omitempty,max=100"`
}

// DeleteTeamRequest 删除团队请求
type DeleteTeamRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// GetTeamRequest 获取团队详情请求
type GetTeamRequest struct {
	ID int64 `form:"id" binding:"required"`
}

// TeamResponse 团队响应
type TeamResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ProjectID   int64   `json:"project_id"`
	Description *string `json:"description"`
	LeaderName  *string `json:"leader_name"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// TeamSimpleResponse 团队简单响应（用于下拉选择）
type TeamSimpleResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ProjectID int64  `json:"project_id"`
}
