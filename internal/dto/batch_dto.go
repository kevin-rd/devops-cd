package dto

import "time"

// BatchListQuery 批次列表查询参数
type BatchListQuery struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
	Status    []int8 `json:"status" form:"status"` // 支持多状态查询
	Initiator string `json:"initiator" form:"initiator"`

	// 新增字段
	ApprovalStatus *string `json:"approval_status" form:"approval_status"`              // pending/approved/rejected
	CreatedAtStart *string `json:"created_at_start" form:"start_time,created_at_start"` // RFC3339格式，例如：2025-01-01T00:00:00Z
	CreatedAtEnd   *string `json:"created_at_end" form:"end_time,created_at_start"`     // RFC3339格式，例如：2025-12-31T23:59:59Z
	Keyword        string  `json:"keyword" form:"keyword"`                              // 模糊搜索批次编号、发起人、发布说明
}

// GetPage 获取页码（默认为1）
func (q *BatchListQuery) GetPage() int {
	if q.Page < 1 {
		return 1
	}
	return q.Page
}

// GetPageSize 获取页大小（默认为20）
func (q *BatchListQuery) GetPageSize() int {
	if q.PageSize < 1 {
		return 20
	}
	if q.PageSize > 100 {
		return 100
	}
	return q.PageSize
}

// BatchGetRequest 获取批次详情请求
type BatchGetRequest struct {
	ID          int64 `json:"id" form:"id" binding:"required"`
	AppPage     int   `json:"app_page" form:"app_page"`           // 应用列表页码，默认1
	AppPageSize int   `json:"app_page_size" form:"app_page_size"` // 应用列表每页数量，默认20
}

// GetAppPage 获取应用页码（默认为1）
func (q *BatchGetRequest) GetAppPage() int {
	if q.AppPage < 1 {
		return 1
	}
	return q.AppPage
}

// GetAppPageSize 获取应用页大小（默认为20，最大50）
func (q *BatchGetRequest) GetAppPageSize() int {
	if q.AppPageSize < 1 {
		return 20
	}
	if q.AppPageSize > 50 {
		return 50
	}
	return q.AppPageSize
}

// BatchResponse 批次响应
type BatchResponse struct {
	// 基本信息
	ID           int64   `json:"id"`
	BatchNumber  string  `json:"batch_number"`
	Initiator    string  `json:"initiator"`
	ReleaseNotes *string `json:"release_notes,omitempty"`

	// 状态信息
	Status         int8   `json:"status"`
	StatusName     string `json:"status_name"`
	ApprovalStatus string `json:"approval_status"`
	AppCount       int64  `json:"app_count"` // 应用数量

	// 审批信息
	ApprovedBy   *string `json:"approved_by,omitempty"`
	ApprovedAt   *string `json:"approved_at,omitempty"`
	RejectReason *string `json:"reject_reason,omitempty"`

	// 时间追踪
	TaggedAt             *string `json:"tagged_at,omitempty"`               // 封板时间
	PreDeployStartedAt   *string `json:"pre_deploy_started_at,omitempty"`   // 预发布开始时间
	PreDeployFinishedAt  *string `json:"pre_deploy_finished_at,omitempty"`  // 预发布完成时间
	ProdDeployStartedAt  *string `json:"prod_deploy_started_at,omitempty"`  // 生产部署开始时间
	ProdDeployFinishedAt *string `json:"prod_deploy_finished_at,omitempty"` // 生产部署完成时间

	// 验收信息
	FinalAcceptedAt *string `json:"final_accepted_at,omitempty"` // 最终验收时间
	FinalAcceptedBy *string `json:"final_accepted_by,omitempty"`

	// 取消信息
	CancelledAt  *string `json:"cancelled_at,omitempty"`
	CancelledBy  *string `json:"cancelled_by,omitempty"`
	CancelReason *string `json:"cancel_reason,omitempty"`

	// 系统字段
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// BatchDetailResponse 批次详情响应（包含应用列表，支持分页）
type BatchDetailResponse struct {
	BatchResponse
	Apps           []ReleaseAppResponse         `json:"apps"`
	TotalApps      int64                        `json:"total_apps"`    // 应用总数
	AppPage        int                          `json:"app_page"`      // 当前页码
	AppPageSize    int                          `json:"app_page_size"` // 每页数量
	AppTypeConfigs map[string]AppTypeConfigInfo `json:"app_type_configs,omitempty"`
}

// AppTypeConfigInfo 应用类型配置（附带依赖关系）
type AppTypeConfigInfo struct {
	Label        string   `json:"label"`
	Description  string   `json:"description,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	Color        string   `json:"color,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// ReleaseAppResponse release_app 响应
type ReleaseAppResponse struct {
	// ReleaseApp 基本信息
	ID      int64  `json:"id"`
	BatchID int64  `json:"batch_id"`
	AppID   int64  `json:"app_id"`
	BuildID *int64 `json:"build_id,omitempty"` // 关联的构建ID

	// 版本信息
	LatestBuildID       *int64  `json:"latest_build_id"`                 // 最新检测到的构建ID（新tag到达时更新）
	PreviousDeployedTag *string `json:"previous_deployed_tag,omitempty"` // 部署前的版本（封板时记录）
	TargetTag           *string `json:"target_tag,omitempty"`            // 目标部署版本（封板时固定，部署期间代表期望版本，部署完成后代表已部署版本）

	// 应用信息
	AppName        string  `json:"app_name"`
	AppDisplayName *string `json:"app_display_name,omitempty"`
	AppType        string  `json:"app_type"`
	AppProject     string  `json:"app_project"`
	AppStatus      int8    `json:"app_status"`
	DeployedTag    *string `json:"deployed_tag,omitempty"` // 应用当前部署的镜像标签（从 applications 表获取）

	// 仓库信息
	RepoID       int64  `json:"repo_id"`
	RepoName     string `json:"repo_name"`
	RepoFullName string `json:"repo_full_name"`

	// 团队信息
	TeamID   *int64  `json:"team_id,omitempty"`
	TeamName *string `json:"team_name,omitempty"`

	// 构建信息（通过 build 关联获取，可选）
	BuildNumber   *int    `json:"build_number,omitempty"` // 构建编号
	BuildStatus   *string `json:"build_status,omitempty"` // 构建状态
	BuildTime     *string `json:"build_time,omitempty"`
	ImageURL      *string `json:"image_url,omitempty"`      // 完整镜像地址
	CommitSHA     *string `json:"commit_sha,omitempty"`     // commit SHA
	CommitMessage *string `json:"commit_message,omitempty"` // commit 信息
	CommitBranch  *string `json:"commit_branch,omitempty"`  // 分支

	// 发布信息
	ReleaseNotes *string `json:"release_notes,omitempty"` // 应用级发布说明
	IsLocked     bool    `json:"is_locked"`               // 是否已锁定（封板后为true）
	Reason       string  `json:"reason,omitempty"`
	Status       int8    `json:"status"`

	// 时间信息
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// 依赖信息
	DefaultDependsOn []int64 `json:"default_depends_on"`
	TempDependsOn    []int64 `json:"temp_depends_on"`

	// 最近的构建记录（自上次部署以来，最多15条）
	RecentBuilds []BuildSummary `json:"recent_builds,omitempty"`
}

// BuildSummary 构建摘要（用于展示自上次部署以来的构建列表）
type BuildSummary struct {
	ID            int64  `json:"id"`
	BuildNumber   int    `json:"build_number"`
	BuildStatus   string `json:"build_status"`
	ImageTag      string `json:"image_tag"`
	CommitSHA     string `json:"commit_sha"`
	CommitMessage string `json:"commit_message"`
	CommitAuthor  string `json:"commit_author"`
	BuildCreated  string `json:"build_created"`
}

// UpdateBuildsRequest 更新批次应用的构建版本请求
type UpdateBuildsRequest struct {
	BatchID      int64           `json:"batch_id" binding:"required"`
	Operator     string          `json:"operator" binding:"required"`
	BuildChanges map[int64]int64 `json:"build_changes" binding:"required"` // key: app_id, value: build_id
}

// UpdateReleaseDependenciesRequest 更新批次应用临时依赖请求
type UpdateReleaseDependenciesRequest struct {
	BatchID       int64   `json:"batch_id" binding:"required"`
	Operator      string  `json:"operator" binding:"required"`
	TempDependsOn []int64 `json:"temp_depends_on"`
	ReleaseAppID  int64   `json:"-"`
}

// ReleaseDependenciesResponse 发布应用依赖响应
type ReleaseDependenciesResponse struct {
	BatchID          int64   `json:"batch_id"`
	ReleaseAppID     int64   `json:"release_app_id"`
	AppID            int64   `json:"app_id"`
	DefaultDependsOn []int64 `json:"default_depends_on"`
	TempDependsOn    []int64 `json:"temp_depends_on"`
	UpdatedAt        string  `json:"updated_at"`
}

// ToBatchResponse 转换为批次响应
func ToBatchResponse(batch interface{}, appCount int64) *BatchResponse {
	// 这个函数会在 service 层使用
	return nil
}

// BatchStatusRequest 获取批次状态请求
type BatchStatusRequest struct {
	ID          int64 `json:"id" form:"id" binding:"required"`
	AppPage     int   `json:"app_page" form:"app_page"`           // 应用列表页码，默认1
	AppPageSize int   `json:"app_page_size" form:"app_page_size"` // 应用列表每页数量，默认20
}

// GetAppPage 获取应用页码（默认为1）
func (q *BatchStatusRequest) GetAppPage() int {
	if q.AppPage < 1 {
		return 1
	}
	return q.AppPage
}

// GetAppPageSize 获取应用页大小（默认为20，最大50）
func (q *BatchStatusRequest) GetAppPageSize() int {
	if q.AppPageSize < 1 {
		return 20
	}
	if q.AppPageSize > 50 {
		return 50
	}
	return q.AppPageSize
}

// BatchStatusResponse 批次状态响应（轻量级，用于状态轮询）
type BatchStatusResponse struct {
	// Batch 基本信息
	ID             int64  `json:"id"`
	BatchNumber    string `json:"batch_number"`
	Status         int8   `json:"status"`
	StatusName     string `json:"status_name"`
	ApprovalStatus string `json:"approval_status"`

	// Batch 时间节点
	SealedAt             *string `json:"sealed_at,omitempty"`
	PreDeployStartedAt   *string `json:"pre_deploy_started_at,omitempty"`
	PreDeployFinishedAt  *string `json:"pre_deploy_finished_at,omitempty"`
	ProdDeployStartedAt  *string `json:"prod_deploy_started_at,omitempty"`
	ProdDeployFinishedAt *string `json:"prod_deploy_finished_at,omitempty"`
	FinalAcceptedAt      *string `json:"final_accepted_at,omitempty"`
	CancelledAt          *string `json:"cancelled_at,omitempty"`
	UpdatedAt            string  `json:"updated_at"`

	// Release Apps 状态列表（不关联其他表）
	Apps        []ReleaseAppStatusResponse `json:"apps"`
	TotalApps   int64                      `json:"total_apps"`
	AppPage     int                        `json:"app_page"`
	AppPageSize int                        `json:"app_page_size"`
}

// ReleaseAppStatusResponse 发布应用状态响应（轻量级，不关联其他表）
type ReleaseAppStatusResponse struct {
	ID            int64          `json:"id"`                        // release_app ID
	AppID         int64          `json:"app_id"`                    // 应用 ID
	Status        int8           `json:"status"`                    // 应用发布状态
	IsLocked      bool           `json:"is_locked"`                 // 是否已锁定
	BuildID       *int64         `json:"build_id,omitempty"`        // 构建 ID
	LatestBuildID *int64         `json:"latest_build_id,omitempty"` // 最新构建 ID
	RecentBuilds  []BuildSummary `json:"recent_builds,omitempty"`   // 最近的构建记录
}

// GetReleaseAppRequest 获取发布应用详情请求
type GetReleaseAppRequest struct {
	ID int64 `form:"id" binding:"required"` // 发布应用ID
}

// FormatTime 格式化时间
func FormatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}
