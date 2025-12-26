package model

import (
	"gorm.io/datatypes"
	"time"
)

const DeploymentTableName = "deployments"

// Deployment 部署记录（批次部署追踪）
type Deployment struct {
	BaseModel

	BatchID   int64 `gorm:"column:batch_id;not null" json:"batch_id"`
	AppID     int64 `gorm:"column:app_id;not null" json:"app_id"`
	ReleaseID int64 `gorm:"column:release_id;not null" json:"release_id"`

	// 部署信息
	Env            string            `gorm:"column:env;size:20;not null" json:"env"` // pre/prod
	ClusterName    string            `gorm:"column:cluster;size:63;not null" json:"cluster_name"`
	Namespace      string            `gorm:"size:63;not null" json:"namespace"`
	DeploymentName string            `gorm:"size:63;not null" json:"deployment_name"`
	Values         datatypes.JSONMap `gorm:"type:json" json:"values"` // 合并后的helm values

	// 状态追踪
	DriverType    *string `gorm:"column:driver_type;size:32" json:"driver_type"`  // main 阶段 driver（如 helm）；为空表示尚未启动 main
	Status        string  `gorm:"size:20;not null;default:pending" json:"status"` // pending/running/success/failed
	RetryCount    int     `gorm:"default:0" json:"retry_count"`
	MaxRetryCount int     `gorm:"default:3" json:"max_retry_count"`

	// 错误信息
	ErrorMessage *string `gorm:"type:text" json:"error_message"`

	// 时间追踪
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`

	// Relations
	Application *Application `gorm:"foreignKey:AppID" json:"application,omitempty"`
	Cluster     *Cluster     `gorm:"foreignKey:ClusterName;references:Name" json:"cluster,omitempty"`
}

// TableName 指定表名
func (Deployment) TableName() string {
	return DeploymentTableName
}
