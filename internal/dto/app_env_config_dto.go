package dto

// CreateAppEnvConfigRequest 创建应用环境配置请求
type CreateAppEnvConfigRequest struct {
	AppID                  int64   `json:"app_id" binding:"required"`
	Env                    string  `json:"env" binding:"required,oneof=pre prod dev test uat"`
	Cluster                string  `json:"cluster" binding:"required,max=50"`
	Replicas               int     `json:"replicas" binding:"required,min=1,max=100"`
	DeploymentNameOverride *string `json:"deployment_name_override" binding:"omitempty,max=63"`
	ConfigData             *string `json:"config_data"` // JSON字符串
}

// UpdateAppEnvConfigRequest 更新应用环境配置请求
type UpdateAppEnvConfigRequest struct {
	ID                     int64   `json:"id" binding:"required"`
	Cluster                *string `json:"cluster" binding:"omitempty,max=50"`
	Replicas               *int    `json:"replicas" binding:"omitempty,min=1,max=100"`
	DeploymentNameOverride *string `json:"deployment_name_override" binding:"omitempty,max=63"`
	ConfigData             *string `json:"config_data"`
	Status                 *int8   `json:"status" binding:"omitempty,oneof=0 1"`
}

// AppEnvConfigResponse 应用环境配置响应
type AppEnvConfigResponse struct {
	ID      int64  `json:"id"`
	AppID   int64  `json:"app_id"`
	Env     string `json:"env"`
	Cluster string `json:"cluster"`

	DeploymentNameOverride *string `json:"deployment_name_override"`
	Replicas               int     `json:"replicas"`
	ConfigData             *string `json:"config_data"`

	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListAppEnvConfigsQuery 查询应用环境配置列表
type ListAppEnvConfigsQuery struct {
	AppID int64   `form:"app_id" binding:"required"`
	Env   *string `form:"env" binding:"omitempty,oneof=pre prod dev test uat"`
}

// DeleteAppEnvConfigRequest 删除应用环境配置请求
type DeleteAppEnvConfigRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// BatchCreateAppEnvConfigsRequest 批量创建应用环境配置请求
type BatchCreateAppEnvConfigsRequest struct {
	AppID   int64                           `json:"app_id" binding:"required"`
	Configs []CreateAppEnvConfigItemRequest `json:"configs" binding:"required,min=1"`
}

// CreateAppEnvConfigItemRequest 批量创建中的单个配置项
type CreateAppEnvConfigItemRequest struct {
	Env                    string  `json:"env" binding:"required,oneof=pre prod dev test uat"`
	Cluster                string  `json:"cluster" binding:"required,max=50"`
	Replicas               int     `json:"replicas" binding:"required,min=1,max=100"`
	DeploymentNameOverride *string `json:"deployment_name_override" binding:"omitempty,max=63"`
	ConfigData             *string `json:"config_data"`
}
