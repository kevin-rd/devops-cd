package api

// GitProvider Git平台提供者接口
type GitProvider interface {
	// TestConnection 测试连接
	TestConnection() error

	// ListRepositories 获取指定所有者的仓库列表
	// owner: 用户名或组织名
	ListRepositories(owner string) ([]RepositoryInfo, error)

	// GetCurrentUser 获取当前认证用户信息（用于 sync_mode=user）
	GetCurrentUser() (*UserInfo, error)

	// ListAllAccessibleRepositories 获取所有可访问的仓库（用于 sync_mode=all）
	ListAllAccessibleRepositories() ([]RepositoryInfo, error)

	// GetPlatformType 获取平台类型
	GetPlatformType() PlatformType
}
