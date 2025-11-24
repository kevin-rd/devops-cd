package dto

// ClusterCreateRequest 创建集群请求
type ClusterCreateRequest struct {
	Name        string  `json:"name" binding:"required,max=50" example:"cluster-prod-01"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=100" example:"生产集群01"`
	Description *string `json:"description" example:"华东区域生产集群"`
	Region      *string `json:"region" binding:"omitempty,max=50" example:"cn-east-1"`
}

// ClusterUpdateRequest 更新集群请求
type ClusterUpdateRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=50" example:"cluster-prod-01"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=100" example:"生产集群01"`
	Description *string `json:"description" example:"华东区域生产集群"`
	Region      *string `json:"region" binding:"omitempty,max=50" example:"cn-east-1"`
	Status      *int8   `json:"status" binding:"omitempty,oneof=0 1" example:"1"`
}

// ClusterResponse 集群响应
type ClusterResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
	Region      *string `json:"region"`
	Status      int8    `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// ClusterListRequest 集群列表请求
type ClusterListRequest struct {
	Name     *string `form:"name" example:"prod"`
	Status   *int8   `form:"status" binding:"omitempty,oneof=0 1" example:"1"`
	Page     int     `form:"page" example:"1"`
	PageSize int     `form:"page_size" example:"10"`
}
