package model

import "time"

// Deployment 部署记录（批次部署追踪）
type Deployment struct {
	ID        int64 `gorm:"primaryKey" json:"id"`
	BatchID   int64 `gorm:"column:batch_id;not null" json:"batch_id"`
	AppID     int64 `gorm:"column:app_id;not null" json:"app_id"`
	ReleaseID int64 `gorm:"column:release_id;not null" json:"release_id"`

	// 部署信息
	DeploymentName string `gorm:"size:100;not null" json:"deployment_name"`
	Environment    string `gorm:"size:20;not null" json:"environment"` // pre/prod
	Cluster        string `gorm:"size:100;not null" json:"cluster"`

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

	// 系统字段
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Deployment) TableName() string {
	return "deployments"
}

// Environment 环境模型
type Environment struct {
	BaseStatus
	Name            string  `gorm:"size:50;not null;uniqueIndex" json:"name"`
	DisplayName     *string `gorm:"size:100" json:"display_name"`
	Description     *string `gorm:"type:text" json:"description"`
	EnvType         string  `gorm:"size:20;not null;index" json:"env_type"`
	ClusterName     *string `gorm:"size:100" json:"cluster_name"`
	ClusterURL      *string `gorm:"size:500" json:"cluster_url"`
	ClusterToken    *string `gorm:"size:1000" json:"-"` // 加密存储,不返回
	Namespace       *string `gorm:"size:100" json:"namespace"`
	Priority        int     `gorm:"not null;default:0;index" json:"priority"`
	RequireApproval bool    `gorm:"not null;default:false" json:"require_approval"`
	AutoDeploy      bool    `gorm:"not null;default:false" json:"auto_deploy"`
}

func (Environment) TableName() string {
	return "environments"
}

// AppEnvConfig 应用环境配置模型
type AppEnvConfig struct {
	BaseStatus

	AppID         int64   `gorm:"column:app_id;not null;uniqueIndex:idx_app_env" json:"app_id"`
	EnvironmentID int64   `gorm:"not null;uniqueIndex:idx_app_env" json:"environment_id"`
	ConfigData    *string `gorm:"type:text" json:"config_data"`
	EnvVars       *string `gorm:"type:text" json:"-"` // 加密存储,不返回
	Replicas      *int    `gorm:"default:1" json:"replicas"`
	CPULimit      *string `gorm:"size:20" json:"cpu_limit"`
	MemoryLimit   *string `gorm:"size:20" json:"memory_limit"`
	CPURequest    *string `gorm:"size:20" json:"cpu_request"`
	MemoryRequest *string `gorm:"size:20" json:"memory_request"`

	// Relations
	Application *Application `gorm:"foreignKey:AppID" json:"application,omitempty"`
	Environment *Environment `gorm:"foreignKey:EnvironmentID" json:"environment,omitempty"`
}

func (AppEnvConfig) TableName() string {
	return "app_env_configs"
}
