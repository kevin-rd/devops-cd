package model

import "time"

const DeploymentTableName = "deployments"

// Deployment 部署记录（批次部署追踪）
type Deployment struct {
	BaseModel

	BatchID   int64 `gorm:"column:batch_id;not null" json:"batch_id"`
	AppID     int64 `gorm:"column:app_id;not null" json:"app_id"`
	ReleaseID int64 `gorm:"column:release_id;not null" json:"release_id"`

	// 部署信息
	Env            string `gorm:"column:env;size:20;not null" json:"env"` // pre/prod
	ClusterName    string `gorm:"column:cluster;size:63;not null" json:"cluster"`
	Namespace      string `gorm:"size:63;not null" json:"namespace"`
	DeploymentName string `gorm:"size:63;not null" json:"deployment_name"`
	ValuesYAML     string `gorm:"type:text" json:"values_yaml"` // 合并后的values.yaml文件

	ImageURL string  `gorm:"size:500" json:"image_url"`
	ImageTag string  `gorm:"size:100" json:"image_tag"`
	TaskID   *string `gorm:"size:100" json:"task_id"` // K8s部署任务ID

	// 状态追踪
	Status        string `gorm:"size:20;not null;default:pending" json:"status"` // pending/running/success/failed
	RetryCount    int    `gorm:"default:0" json:"retry_count"`
	MaxRetryCount int    `gorm:"default:3" json:"max_retry_count"`

	// 错误信息
	ErrorMessage *string `gorm:"type:text" json:"error_message"`

	// 时间追踪
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`

	Application *Application `gorm:"foreignKey:AppID" json:"application,omitempty"`
	Cluster     *Cluster     `gorm:"foreignKey:ClusterName;references:Name" json:"cluster,omitempty"`
}

// TableName 指定表名
func (Deployment) TableName() string {
	return DeploymentTableName
}
