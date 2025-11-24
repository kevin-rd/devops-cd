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

// Cluster 集群元数据模型
// 用途: 管理集群基本信息,不包含连接配置
type Cluster struct {
	BaseStatus
	Name        string  `gorm:"size:50;not null;uniqueIndex" json:"name"`
	DisplayName *string `gorm:"size:100" json:"display_name"`
	Description *string `gorm:"type:text" json:"description"`
	Region      *string `gorm:"size:50" json:"region"`
}

func (Cluster) TableName() string {
	return "clusters"
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
// 用途: 通过记录存在判断应用是否需要部署到某环境,每个集群一条记录
//
// 设计说明:
//   - 有 env='pre' 记录 -> 应用需要部署到 pre
//   - 无 env='pre' 记录 -> 应用跳过 pre,直接到 prod
//   - 每个集群独立配置,支持不同副本数和扩展配置
//
// 示例:
//
//	{app_id: 1, env: "prod", cluster: "cluster-a", replicas: 2}
//	{app_id: 1, env: "prod", cluster: "cluster-b", replicas: 3}
type AppEnvConfig struct {
	BaseStatus

	AppID      int64   `gorm:"column:app_id;not null;uniqueIndex:uk_app_env_cluster" json:"app_id"`
	Env        string  `gorm:"size:20;not null;uniqueIndex:uk_app_env_cluster;index:idx_app_env" json:"env"`   // pre/prod/dev/test/uat
	Cluster    string  `gorm:"size:50;not null;default:default;uniqueIndex:uk_app_env_cluster" json:"cluster"` // 集群名称
	Replicas   int     `gorm:"default:1" json:"replicas"`                                                      // 副本数
	ConfigData *string `gorm:"type:json" json:"config_data,omitempty"`                                         // 扩展配置(JSON格式)

	// Relations
	Application *Application `gorm:"foreignKey:AppID" json:"application,omitempty"`
}

func (AppEnvConfig) TableName() string {
	return "app_env_configs"
}
