package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devops-cd/internal/pkg/git/api"
)

// Provider GitLab平台提供者
type Provider struct {
	config     *api.ProviderConfig
	httpClient *http.Client
}

// NewProvider 创建GitLab提供者
func NewProvider(config *api.ProviderConfig) (api.GitProvider, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL不能为空")
	}

	return &Provider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GetPlatformType 获取平台类型
func (p *Provider) GetPlatformType() api.PlatformType {
	return api.PlatformGitLab
}

// TestConnection 测试连接
func (p *Provider) TestConnection() error {
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := fmt.Sprintf("%s/api/v4/user", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
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

// ListRepositories 获取仓库列表
func (p *Provider) ListRepositories(owner string) ([]api.RepositoryInfo, error) {
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")

	// GitLab使用组或用户的projects端点
	// 先尝试作为用户
	url := fmt.Sprintf("%s/api/v4/users/%s/projects?per_page=100", baseURL, owner)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 尝试作为group
		url = fmt.Sprintf("%s/api/v4/groups/%s/projects?per_page=100", baseURL, owner)
		req, _ = http.NewRequest("GET", url, nil)
		p.setAuthHeader(req)

		resp, err = p.httpClient.Do(req)
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

	repos := make([]api.RepositoryInfo, len(gitlabProjects))
	for i, p := range gitlabProjects {
		repos[i] = api.RepositoryInfo{
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

// GetCurrentUser 获取当前用户
func (p *Provider) GetCurrentUser() (*api.UserInfo, error) {
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := fmt.Sprintf("%s/api/v4/user", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取用户信息失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	var user struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &api.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Name:     user.Name,
	}, nil
}

// ListAllAccessibleRepositories 获取所有可访问的仓库
func (p *Provider) ListAllAccessibleRepositories() ([]api.RepositoryInfo, error) {
	// GitLab可以直接获取当前用户所有可访问的项目
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := fmt.Sprintf("%s/api/v4/projects?membership=true&per_page=100", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败 (状态码: %d): %s", resp.StatusCode, string(body))
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

	repos := make([]api.RepositoryInfo, len(gitlabProjects))
	for i, p := range gitlabProjects {
		repos[i] = api.RepositoryInfo{
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

// setAuthHeader 设置认证头
func (p *Provider) setAuthHeader(req *http.Request) {
	if p.config.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", p.config.Token)
	}
}
