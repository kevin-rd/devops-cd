package git

import (
	"devops-cd/internal/pkg/git/api"
	"devops-cd/internal/pkg/git/gitea"
	"devops-cd/internal/pkg/git/github"
	"devops-cd/internal/pkg/git/gitlab"
	"fmt"
)

// ClientConfig 客户端配置（向后兼容）
type ClientConfig struct {
	BaseURL      string           // 平台基础URL，如: https://gitea.com
	Token        string           // 访问Token
	PlatformType api.PlatformType // 平台类型
}

// Client Git平台客户端（向后兼容的包装器）
type Client struct {
	provider api.GitProvider
}

// NewClient 创建Git平台客户端（工厂方法）
func NewClient(baseUrl, token, platform string) (*Client, error) {
	if baseUrl == "" {
		return nil, fmt.Errorf("BaseURL不能为空")
	}

	providerConfig := &api.ProviderConfig{
		BaseURL: baseUrl,
		Token:   token,
	}

	var provider api.GitProvider
	var err error

	// 根据平台类型创建对应的提供者
	switch platform {
	case string(api.PlatformGitea):
		provider, err = gitea.NewProvider(providerConfig)
	case string(api.PlatformGitLab):
		provider, err = gitlab.NewProvider(providerConfig)
	case string(api.PlatformGitHub):
		provider, err = github.NewProvider(providerConfig)
	default:
		return nil, fmt.Errorf("不支持的平台类型: %s", platform)
	}

	if err != nil {
		return nil, err
	}

	return &Client{provider: provider}, nil
}

// TestConnection 测试连接
func (c *Client) TestConnection() error {
	return c.provider.TestConnection()
}

// ListRepositories 获取仓库列表
func (c *Client) ListRepositories(owner string) ([]api.RepositoryInfo, error) {
	return c.provider.ListRepositories(owner)
}

// GetCurrentUser 获取当前用户
func (c *Client) GetCurrentUser() (*api.UserInfo, error) {
	return c.provider.GetCurrentUser()
}

// ListAllAccessibleRepositories 获取所有可访问的仓库
func (c *Client) ListAllAccessibleRepositories() ([]api.RepositoryInfo, error) {
	return c.provider.ListAllAccessibleRepositories()
}

// GetProvider 获取底层提供者（供高级使用）
func (c *Client) GetProvider() api.GitProvider {
	return c.provider
}
