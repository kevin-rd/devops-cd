package dto

import "encoding/json"

// CreateCredentialRequest 创建凭据请求
// data: 敏感字段（按 type 不同而不同），将被加密后存储；服务端不会回传明文
type CreateCredentialRequest struct {
	Scope     string          `json:"scope" binding:"required,oneof=global project"`
	ProjectID *int64          `json:"project_id"`
	Name      string          `json:"name" binding:"required,max=128"`
	Type      string          `json:"type" binding:"required,oneof=basic_auth token ssh_key tls_client_cert"`
	Data      json.RawMessage `json:"data" binding:"required"` // 加密存储
	Meta      json.RawMessage `json:"meta"`                    // 非敏感展示信息（可选）
}

type UpdateCredentialRequest struct {
	Name string          `json:"name" binding:"required,max=128"`
	Data json.RawMessage `json:"data"` // 可选，更新密文
	Meta json.RawMessage `json:"meta"` // 可选
}

type CredentialResponse struct {
	ID        int64           `json:"id"`
	Scope     string          `json:"scope"`
	ProjectID *int64          `json:"project_id,omitempty"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Meta      json.RawMessage `json:"meta_json,omitempty"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}
