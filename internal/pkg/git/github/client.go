package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devops-cd/internal/pkg/git/api"
)

// Provider GitHub平台提供者
type Provider struct {
	config     *api.ProviderConfig
	httpClient *http.Client
}

// NewProvider 创建GitHub提供者
func NewProvider(config *api.ProviderConfig) (api.GitProvider, error) {
	// GitHub可以省略BaseURL，使用默认值
	if config.BaseURL == "" {
		config.BaseURL = "https://api.github.com"
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
	return api.PlatformGitHub
}

// TestConnection 测试连接
func (p *Provider) TestConnection() error {
	url := "https://api.github.com/user"

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
	// 先尝试用户仓库
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", owner)

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
		// 尝试组织仓库
		url = fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100", owner)
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

	repos := make([]api.RepositoryInfo, len(githubRepos))
	for i, r := range githubRepos {
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

// GetCurrentUser 获取当前用户
func (p *Provider) GetCurrentUser() (*api.UserInfo, error) {
	url := "https://api.github.com/user"

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
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &api.UserInfo{
		ID:       user.ID,
		Username: user.Login,
		Email:    user.Email,
		Name:     user.Name,
	}, nil
}

// ListAllAccessibleRepositories 获取所有可访问的仓库
func (p *Provider) ListAllAccessibleRepositories() ([]api.RepositoryInfo, error) {
	// GitHub可以直接获取当前用户所有可访问的仓库
	url := "https://api.github.com/user/repos?per_page=100&affiliation=owner,collaborator,organization_member"

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

	repos := make([]api.RepositoryInfo, len(githubRepos))
	for i, r := range githubRepos {
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
func (p *Provider) setAuthHeader(req *http.Request) {
	if p.config.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", p.config.Token))
	}
}
