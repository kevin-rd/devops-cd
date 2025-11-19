package api

// PlatformType 平台类型
type PlatformType string

const (
	PlatformGitea  PlatformType = "gitea"
	PlatformGitLab PlatformType = "gitlab"
	PlatformGitHub PlatformType = "github"
)

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

// UserInfo 用户信息
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

// ProviderConfig 通用平台配置
type ProviderConfig struct {
	BaseURL string // 平台基础URL
	Token   string // 访问Token
}
