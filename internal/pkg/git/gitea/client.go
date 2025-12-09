package gitea

import (
	pkgErrors "devops-cd/pkg/responses"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devops-cd/internal/pkg/git/api"
)

// Provider Gitea平台提供者
type Provider struct {
	config     *api.ProviderConfig
	httpClient *http.Client
}

// NewProvider 创建Gitea提供者
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
	return api.PlatformGitea
}

// TestConnection 测试连接
func (p *Provider) TestConnection() error {
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := fmt.Sprintf("%s/api/v1/user", baseURL)

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

	// 先尝试作为用户
	url := fmt.Sprintf("%s/api/v1/users/%s/repos?limit=100", baseURL, owner)
	repos, err := p.fetchRepos(url)
	if err == nil && len(repos) > 0 {
		return repos, nil
	}

	// 尝试作为组织
	url = fmt.Sprintf("%s/api/v1/orgs/%s/repos?limit=100", baseURL, owner)
	repos, err = p.fetchRepos(url)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "获取Gitea仓库失败", err)
	}

	return repos, nil
}

// GetCurrentUser 获取当前用户
func (p *Provider) GetCurrentUser() (*api.UserInfo, error) {
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := fmt.Sprintf("%s/api/v1/user", baseURL)

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
		Login    string `json:"login"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &api.UserInfo{
		ID:       user.ID,
		Username: user.Login,
		Email:    user.Email,
		Name:     user.FullName,
	}, nil
}

// ListAllAccessibleRepositories 获取所有可访问的仓库
func (p *Provider) ListAllAccessibleRepositories() ([]api.RepositoryInfo, error) {
	// Gitea没有直接获取所有仓库的API
	// 需要先获取当前用户，然后获取用户的仓库
	user, err := p.GetCurrentUser()
	if err != nil {
		return nil, fmt.Errorf("获取当前用户失败: %w", err)
	}

	return p.ListRepositories(user.Username)
}

// fetchRepos 获取仓库列表
func (p *Provider) fetchRepos(url string) ([]api.RepositoryInfo, error) {
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

	repos := make([]api.RepositoryInfo, len(giteaRepos))
	for i, r := range giteaRepos {
		repos[i] = api.RepositoryInfo{
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

// setAuthHeader 设置认证头
func (p *Provider) setAuthHeader(req *http.Request) {
	if p.config.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", p.config.Token))
	}
}
