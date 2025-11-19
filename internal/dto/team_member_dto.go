package dto

// TeamMemberAddRequest 添加成员请求
type TeamMemberAddRequest struct {
	TeamID int64   `json:"team_id" binding:"required"`
	UserID int64   `json:"user_id" binding:"required"`
	Role   *string `json:"role" binding:"omitempty,max=20"`
}

// TeamMemberUpdateRoleRequest 更新成员角色
type TeamMemberUpdateRoleRequest struct {
	Role string `json:"role" binding:"required,max=20"`
}

// TeamMemberListQuery 成员列表请求
type TeamMemberListQuery struct {
	PageQuery
	TeamID int64 `form:"team_id" binding:"required"`
}

// TeamMemberResponse 成员响应
type TeamMemberResponse struct {
	ID          int64   `json:"id"`
	TeamID      int64   `json:"team_id"`
	UserID      int64   `json:"user_id"`
	Role        string  `json:"role"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
	Email       *string `json:"email"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}
