package model

import "gorm.io/datatypes"

const CredentialTableName = "config_credentials"

type CredentialType string

const (
	CredentialTypeBasicAuth     CredentialType = "basic_auth"      // username/password
	CredentialTypeToken         CredentialType = "token"           // bearer/token
	CredentialTypeSSHKey        CredentialType = "ssh_key"         // private key (+ optional passphrase)
	CredentialTypeTLSClientCert CredentialType = "tls_client_cert" // client cert/key (+ optional ca)
)

// Credential 凭据（敏感字段加密存储）
//
// 说明：
// - encrypted_data: AES-GCM(base64) 密文（nonce 已包含在密文中）
// - meta_json: 非敏感字段（用于列表展示/筛选）
type Credential struct {
	BaseModelWithSoftDelete

	Scope     Scope  `gorm:"size:16;not null;index" json:"scope"` // global/project
	ProjectID *int64 `gorm:"column:project_id;index" json:"project_id,omitempty"`
	Name      string `gorm:"size:128;not null" json:"name"`
	Type      string `gorm:"size:32;not null" json:"type"`

	EncryptedData string         `gorm:"column:encrypted_data;type:longtext;not null" json:"-"`
	MetaJSON      datatypes.JSON `gorm:"column:meta_json;type:json" json:"meta_json,omitempty"`
}

func (Credential) TableName() string {
	return CredentialTableName
}
