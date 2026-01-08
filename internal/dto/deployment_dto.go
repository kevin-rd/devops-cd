package dto

// RetryDeploymentRequest 手动重试部署请求
type RetryDeploymentRequest struct {
	Operator string `json:"operator" binding:"required"` // 操作人
	Reason   string `json:"reason"`                      // 重试原因（可选）
}
