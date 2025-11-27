package dto

// SwitchVersionRequest 切换版本请求
type SwitchVersionRequest struct {
	BatchID      int64  `json:"batch_id" binding:"required"`       // 批次ID
	ReleaseAppID int64  `json:"release_app_id" binding:"required"` // 发布应用ID
	Operator     string `json:"operator" binding:"required"`       // 操作人
	BuildID      int64  `json:"build_id" binding:"required"`       // 目标build id
	Reason       string `json:"reason"`                            // 触发原因（可选）
}

// ManualDeployRequest 手动部署请求
type ManualDeployRequest struct {
	BatchID      int64  `json:"batch_id" binding:"required"`       // 批次ID
	ReleaseAppID int64  `json:"release_app_id" binding:"required"` // 发布应用ID
	Action       string `json:"action" binding:"required"`         // 部署环境: pre/prod
	Operator     string `json:"operator" binding:"required"`       // 操作人
	Reason       string `json:"reason"`                            // 触发原因（可选）
}
