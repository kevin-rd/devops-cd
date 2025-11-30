package model

const ClusterTableName = "clusters"
const AppEnvConfigTableName = "app_env_configs"

// Cluster 集群元数据模型
// 用途: 管理集群基本信息,不包含连接配置
type Cluster struct {
	BaseStatus
	Name        string  `gorm:"size:50;not null;uniqueIndex" json:"name"`
	Description *string `gorm:"type:text" json:"description"`
	Region      *string `gorm:"size:50" json:"region"`
}

func (Cluster) TableName() string {
	return ClusterTableName
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
	return AppEnvConfigTableName
}
