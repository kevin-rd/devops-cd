package dto

// BuildNotifyRequest 构建通知请求（来自 Drone webhook）
type BuildNotifyRequest struct {
	// ========== 仓库信息 ==========
	Repo          string `json:"repo" binding:"required"`      // zkme/zkme-kyb
	RepoNamespace string `json:"repo_namespace"`               // zkme (可选，兼容 Drone)
	RepoOwner     string `json:"repo_owner"`                   // zkme (可选，兼容 Drone)
	RepoName      string `json:"repo_name" binding:"required"` // zkme-kyb

	// ========== 构建信息 ==========
	BuildNumber   int64  `json:"build_number" binding:"required"`                                             // 11
	BuildStatus   string `json:"build_status" binding:"required,oneof=success failure error killed"`          // 构建状态
	BuildCreated  int64  `json:"build_created" binding:"required"`                                            // Unix timestamp (秒)
	BuildStarted  int64  `json:"build_started" binding:"required"`                                            // Unix timestamp (秒)
	BuildFinished int64  `json:"build_finished" binding:"required"`                                           // Unix timestamp (秒)
	BuildLink     string `json:"build_link" binding:"required,url"`                                           // 构建链接
	BuildEvent    string `json:"build_event" binding:"required,oneof=push tag pull_request promote rollback"` // 触发事件

	// ========== Git 提交信息 ==========
	// Drone 原始字段（保留以兼容）
	GitAuthorName     string `json:"git_author_name"`     // 原作者（可选）
	GitAuthorEmail    string `json:"git_author_email"`    // 原作者邮箱（可选）
	CommitAuthor      string `json:"commit_author"`       // 提交者用户名（可选）
	CommitAuthorName  string `json:"commit_author_name"`  // 提交者名称
	CommitAuthorEmail string `json:"commit_author_email"` // 提交者邮箱

	CommitRef     string  `json:"commit_ref" binding:"required"`       // refs/tags/v2025.1027.01-ga
	CommitID      string  `json:"commit_id" binding:"required"`        // e8c7e9bfe6fb09a7fa1a599b591993c1ab8da47d
	CommitBranch  string  `json:"commit_branch"`                       // branch_dev@zkme-kyb-popup-api
	CommitBefore  *string `json:"commit_before"`                       // 可选：前一个 commit
	CommitAfter   string  `json:"commit_after" binding:"required"`     // e8c7e9bfe6fb09a7fa1a599b591993c1ab8da47d
	CommitMessage string  `json:"commit_message"`                      // 新增测试环境配置文件
	CommitLink    string  `json:"commit_link" binding:"omitempty,url"` // 提交链接（可选）

	// ========== 应用列表 ==========
	Apps []BuildNotifyApp `json:"apps" binding:"required,min=1,dive"` // 至少一个应用
}

// BuildNotifyApp 构建通知中的应用信息
type BuildNotifyApp struct {
	Name         string  `json:"name" binding:"required"`      // zkme-kyb-admin (应用名称，不是image_tag)
	ImageTag     string  `json:"image_tag" binding:"required"` // v2025.1027.01-ga (原字段名)
	Image        *string `json:"image"`                        // 可选：完整镜像地址
	BuildSuccess *bool   `json:"build_success"`                // 可选：该应用是否构建成功
}

// BuildResponse 构建记录响应
type BuildResponse struct {
	ID       int64   `json:"id"`
	RepoID   int64   `json:"repo_id"`
	RepoName *string `json:"repo_name,omitempty"` // 仓库名称（关联查询）
	AppID    int64   `json:"app_id"`
	AppName  *string `json:"app_name,omitempty"` // 应用名称（关联查询）

	BuildNumber int    `json:"build_number"`
	BuildStatus string `json:"build_status"`
	BuildEvent  string `json:"build_event"`
	BuildLink   string `json:"build_link"`

	CommitSHA     string `json:"commit_sha"`
	CommitRef     string `json:"commit_ref"`
	CommitBranch  string `json:"commit_branch"`
	CommitMessage string `json:"commit_message"`
	CommitLink    string `json:"commit_link"`
	CommitAuthor  string `json:"commit_author"`

	BuildCreated  string `json:"build_created"`  // RFC3339 格式时间字符串
	BuildStarted  string `json:"build_started"`  // RFC3339 格式时间字符串
	BuildFinished string `json:"build_finished"` // RFC3339 格式时间字符串
	BuildDuration int    `json:"build_duration"` // 构建耗时（秒）

	ImageTag        string `json:"image_tag"`
	ImageURL        string `json:"image_url"`
	AppBuildSuccess bool   `json:"app_build_success"`
	Environment     string `json:"environment"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// BuildListQuery 构建列表查询参数
type BuildListQuery struct {
	PageQuery           // 分页参数
	RepoID      *int64  `form:"repo_id"`                                                                      // 按仓库筛选
	AppID       *int64  `form:"app_id"`                                                                       // 按应用筛选
	BuildStatus *string `form:"build_status" binding:"omitempty,oneof=success failure error killed"`          // 按状态筛选
	BuildEvent  *string `form:"build_event" binding:"omitempty,oneof=push tag pull_request promote rollback"` // 按事件筛选
	ImageTag    *string `form:"image_tag"`                                                                    // 按镜像标签筛选
	CommitSHA   *string `form:"commit_sha"`                                                                   // 按 commit 查询
	Environment *string `form:"environment"`                                                                  // 按环境筛选
}

// GetBuildRequest 获取单个构建记录请求
type GetBuildRequest struct {
	ID int64 `form:"id" binding:"required"` // 构建记录ID
}

// GetBuildByAppAndNumberRequest 根据应用和构建号查询
type GetBuildByAppAndNumberRequest struct {
	AppID       int64 `form:"app_id" binding:"required"`       // 应用ID
	BuildNumber int   `form:"build_number" binding:"required"` // 构建号
}
