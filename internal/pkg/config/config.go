package config

import (
	"fmt"
	"sort"

	"github.com/spf13/viper"
)

var GlobalConfig *Config

// Config 全局配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Crypto   CryptoConfig   `mapstructure:"crypto"`
	Log      LogConfig      `mapstructure:"log"`
	Core     CoreConfig     `mapstructure:"core"`
	Repo     RepoConfig     `mapstructure:"repo"`
	DB       interface{}    // 数据库连接,运行时注入
}

// ServerConfig 服务配置
type ServerConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 秒
	LogLevel        string `mapstructure:"log_level"`         // SQL日志级别: silent/error/warn/info
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWT   JWTConfig   `mapstructure:"jwt"`
	LDAP  LDAPConfig  `mapstructure:"ldap"`
	Local LocalConfig `mapstructure:"local"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	AccessTokenExpire  int    `mapstructure:"access_token_expire"`  // 秒
	RefreshTokenExpire int    `mapstructure:"refresh_token_expire"` // 秒
}

// LDAPConfig LDAP配置
type LDAPConfig struct {
	Enabled      bool           `mapstructure:"enabled"`
	Host         string         `mapstructure:"host"`
	Port         int            `mapstructure:"port"`
	UseSSL       bool           `mapstructure:"use_ssl"`
	BindDN       string         `mapstructure:"bind_dn"`
	BindPassword string         `mapstructure:"bind_password"`
	BaseDN       string         `mapstructure:"base_dn"`
	UserFilter   string         `mapstructure:"user_filter"`
	Attributes   LDAPAttributes `mapstructure:"attributes"`
}

// LDAPAttributes LDAP属性映射
type LDAPAttributes struct {
	Username    string `mapstructure:"username"`
	Email       string `mapstructure:"email"`
	DisplayName string `mapstructure:"display_name"`
}

// LocalConfig 本地用户配置
type LocalConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// CryptoConfig 加密配置
type CryptoConfig struct {
	AESKey string `mapstructure:"aes_key"` // 32字节
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`  // debug, info, warn, error
	Format     string `mapstructure:"format"` // json, console
	Output     string `mapstructure:"output"` // stdout, file
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
}

// CoreConfig Core模块配置
type CoreConfig struct {
	ScanInterval string                   `mapstructure:"scan_interval"` // 扫描间隔
	Deploy       DeployConfig             `mapstructure:"deploy"`
	Notification NotificationConfig       `mapstructure:"notification"`
	AppTypes     map[string]AppTypeConfig `mapstructure:"app_types"`
}

// DeployConfig 部署配置
type DeployConfig struct {
	ConcurrentApps   int    `mapstructure:"concurrent_apps"`    // 并发部署数
	SingleAppTimeout string `mapstructure:"single_app_timeout"` // 单应用超时
	BatchTimeout     string `mapstructure:"batch_timeout"`      // 批次超时
	RetryCount       int    `mapstructure:"retry_count"`        // 重试次数
	RetryBackoff     string `mapstructure:"retry_backoff"`      // 重试策略
	PollInterval     string `mapstructure:"poll_interval"`      // 轮询间隔
}

// AppTypeConfig 应用类型配置
type AppTypeConfig struct {
	Label        string   `mapstructure:"label"`
	Description  string   `mapstructure:"description"`
	Icon         string   `mapstructure:"icon"`
	Color        string   `mapstructure:"color"`
	Dependencies []string `mapstructure:"dependencies"`
}

// AppTypeMetadata 应用类型元数据
type AppTypeMetadata struct {
	Value        string
	Label        string
	Description  string
	Icon         string
	Color        string
	Dependencies []string
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Enabled     bool   `mapstructure:"enabled"`      // 是否启用
	Provider    string `mapstructure:"provider"`     // 通知渠道
	LarkWebhook string `mapstructure:"lark_webhook"` // Lark Webhook
}

// RepoConfig 代码库同步配置
type RepoConfig struct {
	Cron    string             `mapstructure:"cron"` // Cron表达式，定义同步执行时间
	Sources []RepoSourceConfig `mapstructure:"sources"`
}

// RepoSourceConfig 代码库源配置
type RepoSourceConfig struct {
	Name       string   `mapstructure:"name"`       // 源名称
	Platform   string   `mapstructure:"platform"`   // 平台类型: gitea/gitlab/github
	BaseURL    string   `mapstructure:"base_url"`   // 平台地址
	Token      string   `mapstructure:"token"`      // 访问令牌
	SyncMode   string   `mapstructure:"sync_mode"`  // 同步模式: namespaces/all/user
	Namespaces []string `mapstructure:"namespaces"` // 命名空间列表(sync_mode=namespaces时使用)
	Enabled    bool     `mapstructure:"enabled"`    // 是否启用
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 读取环境变量
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 设置全局配置
	GlobalConfig = config

	return config, nil
}

// GetDSN 获取数据库DSN
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// GetAppTypeConfigs 返回应用类型配置快照
func GetAppTypeConfigs() map[string]AppTypeConfig {
	if GlobalConfig == nil {
		return map[string]AppTypeConfig{}
	}

	result := make(map[string]AppTypeConfig, len(GlobalConfig.Core.AppTypes))
	for key, cfg := range GlobalConfig.Core.AppTypes {
		cloned := cfg
		if cloned.Dependencies == nil {
			cloned.Dependencies = []string{}
		} else {
			deps := make([]string, len(cloned.Dependencies))
			copy(deps, cloned.Dependencies)
			cloned.Dependencies = deps
		}
		result[key] = cloned
	}

	return result
}

// GetAppTypeMetadata 返回应用类型元数据列表（包含配置与依赖）
func GetAppTypeMetadata() []AppTypeMetadata {
	configs := GetAppTypeConfigs()
	metadata := make([]AppTypeMetadata, 0, len(configs))

	for value, cfg := range configs {
		label := cfg.Label
		if label == "" {
			label = value
		}

		deps := cfg.Dependencies
		if deps == nil {
			deps = []string{}
		}

		metadata = append(metadata, AppTypeMetadata{
			Value:        value,
			Label:        label,
			Description:  cfg.Description,
			Icon:         cfg.Icon,
			Color:        cfg.Color,
			Dependencies: deps,
		})
	}

	sort.Slice(metadata, func(i, j int) bool {
		if metadata[i].Label == metadata[j].Label {
			return metadata[i].Value < metadata[j].Value
		}
		return metadata[i].Label < metadata[j].Label
	})

	return metadata
}

// GetAppTypeLabel 获取应用类型的显示名称
func GetAppTypeLabel(appType string) string {
	configs := GetAppTypeConfigs()
	if cfg, ok := configs[appType]; ok {
		if cfg.Label != "" {
			return cfg.Label
		}
	}
	return appType
}

// GetAppTypeDependencies 返回应用类型依赖关系
func GetAppTypeDependencies() map[string][]string {
	configs := GetAppTypeConfigs()
	result := make(map[string][]string, len(configs))
	for value, cfg := range configs {
		deps := make([]string, 0, len(cfg.Dependencies))
		deps = append(deps, cfg.Dependencies...)
		result[value] = deps
	}
	return result
}
