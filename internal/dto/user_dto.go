package dto

// UserSearchQuery 用户搜索请求
type UserSearchQuery struct {
	PageQuery
}

// UserSimpleResponse 用户精简信息
type UserSimpleResponse struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	Email       *string `json:"email,omitempty"`
}
