package git

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	pkgErrors "devops-cd/pkg/errors"
)

// PlatformType 平台类型
type PlatformType string

const (
	PlatformGitea  PlatformType = "gitea"
	PlatformGitLab PlatformType = "gitlab"
	PlatformGitHub PlatformType = "github"
)

// ClientConfig 客户端配置
type ClientConfig struct {
	BaseURL      string       // 平台基础URL，如: https://gitea.com
	Token        string       // 访问Token
	PlatformType PlatformType // 平台类型
}

// Client Git平台客户端
type Client struct {
	config     *ClientConfig
	httpClient *http.Client
}

// RepositoryInfo 仓库信息
type RepositoryInfo struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	FullName      string  `json:"full_name"`
	Description   string  `json:"description"`
	CloneURL      string  `json:"clone_url"`
	SSHURL        string  `json:"ssh_url"`
	DefaultBranch string  `json:"default_branch"`
	Language      string  `json:"language"`
	Private       bool    `json:"private"`
	Fork          bool    `json:"fork"`
	Archived      bool    `json:"archived"`
	Stars         int     `json:"stars"`
	Forks         int     `json:"forks"`
	OpenIssues    int     `json:"open_issues"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	Owner         string  `json:"owner"`
	OwnerType     string  `json:"owner_type"` // user, organization
	HTMLURL       *string `json:"html_url"`
}

// NewClient 创建Git平台客户端
func NewClient(config *ClientConfig) (*Client, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL不能为空")
	}
	if config.PlatformType == "" {
		return nil, fmt.Errorf("PlatformType不能为空")
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// TestConnection 测试连接
func (c *Client) TestConnection() error {
	var url string
	switch c.config.PlatformType {
	case PlatformGitea:
		url = fmt.Sprintf("%s/api/v1/user", strings.TrimSuffix(c.config.BaseURL, "/"))
	case PlatformGitLab:
		url = fmt.Sprintf("%s/api/v4/user", strings.TrimSuffix(c.config.BaseURL, "/"))
	case PlatformGitHub:
		url = "https://api.github.com/user"
	default:
		return fmt.Errorf("不支持的平台类型: %s", c.config.PlatformType)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("连接失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListRepositories 获取用户或组织的仓库列表
func (c *Client) ListRepositories(owner string) ([]RepositoryInfo, error) {
	switch c.config.PlatformType {
	case PlatformGitea:
		return c.listGiteaRepositories(owner)
	case PlatformGitLab:
		return c.listGitLabRepositories(owner)
	case PlatformGitHub:
		return c.listGitHubRepositories(owner)
	default:
		return nil, fmt.Errorf("不支持的平台类型: %s", c.config.PlatformType)
	}
}

// listGiteaRepositories 获取Gitea仓库列表
func (c *Client) listGiteaRepositories(owner string) ([]RepositoryInfo, error) {
	baseURL := strings.TrimSuffix(c.config.BaseURL, "/")

	// 先尝试作为用户
	url := fmt.Sprintf("%s/api/v1/users/%s/repos?limit=100", baseURL, owner)
	repos, err := c.fetchGiteaRepos(url)
	if err == nil && len(repos) > 0 {
		return repos, nil
	}

	// 尝试作为组织
	url = fmt.Sprintf("%s/api/v1/orgs/%s/repos?limit=100", baseURL, owner)
	repos, err = c.fetchGiteaRepos(url)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "获取Gitea仓库失败", err)
	}

	return repos, nil
}

// fetchGiteaRepos 获取Gitea仓库
func (c *Client) fetchGiteaRepos(url string) ([]RepositoryInfo, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	var giteaRepos []struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		CloneURL      string `json:"clone_url"`
		SSHURL        string `json:"ssh_url"`
		DefaultBranch string `json:"default_branch"`
		Language      string `json:"language"`
		Private       bool   `json:"private"`
		Fork          bool   `json:"fork"`
		Archived      bool   `json:"archived"`
		Stars         int    `json:"stars_count"`
		Forks         int    `json:"forks_count"`
		OpenIssues    int    `json:"open_issues_count"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
		HTMLURL       string `json:"html_url"`
		Owner         struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"owner"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&giteaRepos); err != nil {
		return nil, err
	}

	repos := make([]RepositoryInfo, len(giteaRepos))
	for i, r := range giteaRepos {
		repos[i] = RepositoryInfo{
			ID:            r.ID,
			Name:          r.Name,
			FullName:      r.FullName,
			Description:   r.Description,
			CloneURL:      r.CloneURL,
			SSHURL:        r.SSHURL,
			DefaultBranch: r.DefaultBranch,
			Language:      r.Language,
			Private:       r.Private,
			Fork:          r.Fork,
			Archived:      r.Archived,
			Stars:         r.Stars,
			Forks:         r.Forks,
			OpenIssues:    r.OpenIssues,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
			Owner:         r.Owner.Login,
			OwnerType:     strings.ToLower(r.Owner.Type),
			HTMLURL:       &r.HTMLURL,
		}
	}

	return repos, nil
}

// listGitLabRepositories 获取GitLab仓库列表
func (c *Client) listGitLabRepositories(owner string) ([]RepositoryInfo, error) {
	baseURL := strings.TrimSuffix(c.config.BaseURL, "/")

	// GitLab使用组或用户的projects端点
	url := fmt.Sprintf("%s/api/v4/users/%s/projects?per_page=100", baseURL, owner)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 尝试作为group
		url = fmt.Sprintf("%s/api/v4/groups/%s/projects?per_page=100", baseURL, owner)
		req, _ = http.NewRequest("GET", url, nil)
		c.setAuthHeader(req)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("请求失败 (状态码: %d): %s", resp.StatusCode, string(body))
		}
	}

	var gitlabProjects []struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
		SSHURLToRepo      string `json:"ssh_url_to_repo"`
		DefaultBranch     string `json:"default_branch"`
		Visibility        string `json:"visibility"`
		ForkedFromProject *struct {
			ID int64 `json:"id"`
		} `json:"forked_from_project"`
		Archived        bool   `json:"archived"`
		StarCount       int    `json:"star_count"`
		ForksCount      int    `json:"forks_count"`
		OpenIssuesCount int    `json:"open_issues_count"`
		CreatedAt       string `json:"created_at"`
		LastActivityAt  string `json:"last_activity_at"`
		WebURL          string `json:"web_url"`
		Namespace       struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		} `json:"namespace"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gitlabProjects); err != nil {
		return nil, err
	}

	repos := make([]RepositoryInfo, len(gitlabProjects))
	for i, p := range gitlabProjects {
		repos[i] = RepositoryInfo{
			ID:            p.ID,
			Name:          p.Name,
			FullName:      p.PathWithNamespace,
			Description:   p.Description,
			CloneURL:      p.HTTPURLToRepo,
			SSHURL:        p.SSHURLToRepo,
			DefaultBranch: p.DefaultBranch,
			Private:       p.Visibility == "private",
			Fork:          p.ForkedFromProject != nil,
			Archived:      p.Archived,
			Stars:         p.StarCount,
			Forks:         p.ForksCount,
			OpenIssues:    p.OpenIssuesCount,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.LastActivityAt,
			Owner:         p.Namespace.Path,
			OwnerType:     p.Namespace.Kind,
			HTMLURL:       &p.WebURL,
		}
	}

	return repos, nil
}

// listGitHubRepositories 获取GitHub仓库列表
func (c *Client) listGitHubRepositories(owner string) ([]RepositoryInfo, error) {
	// 先尝试用户仓库
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", owner)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 尝试组织仓库
		url = fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100", owner)
		req, _ = http.NewRequest("GET", url, nil)
		c.setAuthHeader(req)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("请求失败 (状态码: %d): %s", resp.StatusCode, string(body))
		}
	}

	var githubRepos []struct {
		ID              int64  `json:"id"`
		Name            string `json:"name"`
		FullName        string `json:"full_name"`
		Description     string `json:"description"`
		CloneURL        string `json:"clone_url"`
		SSHURL          string `json:"ssh_url"`
		DefaultBranch   string `json:"default_branch"`
		Language        string `json:"language"`
		Private         bool   `json:"private"`
		Fork            bool   `json:"fork"`
		Archived        bool   `json:"archived"`
		StargazersCount int    `json:"stargazers_count"`
		ForksCount      int    `json:"forks_count"`
		OpenIssuesCount int    `json:"open_issues_count"`
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
		HTMLURL         string `json:"html_url"`
		Owner           struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"owner"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&githubRepos); err != nil {
		return nil, err
	}

	repos := make([]RepositoryInfo, len(githubRepos))
	for i, r := range githubRepos {
		repos[i] = RepositoryInfo{
			ID:            r.ID,
			Name:          r.Name,
			FullName:      r.FullName,
			Description:   r.Description,
			CloneURL:      r.CloneURL,
			SSHURL:        r.SSHURL,
			DefaultBranch: r.DefaultBranch,
			Language:      r.Language,
			Private:       r.Private,
			Fork:          r.Fork,
			Archived:      r.Archived,
			Stars:         r.StargazersCount,
			Forks:         r.ForksCount,
			OpenIssues:    r.OpenIssuesCount,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
			Owner:         r.Owner.Login,
			OwnerType:     strings.ToLower(r.Owner.Type),
			HTMLURL:       &r.HTMLURL,
		}
	}

	return repos, nil
}

// setAuthHeader 设置认证头
func (c *Client) setAuthHeader(req *http.Request) {
	if c.config.Token == "" {
		return
	}

	switch c.config.PlatformType {
	case PlatformGitea:
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.config.Token))
	case PlatformGitLab:
		req.Header.Set("PRIVATE-TOKEN", c.config.Token)
	case PlatformGitHub:
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.config.Token))
	}
}
