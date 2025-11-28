package dto

import (
	"time"
)

// BatchListQuery 批次列表查询参数
type BatchListQuery struct {
	Page      int     `json:"page" form:"page"`
	PageSize  int     `json:"page_size" form:"page_size"`
	Statuses  []int8  `json:"status" form:"status"` // 支持多状态查询
	Initiator *string `json:"initiator" form:"initiator"`

	// 新增字段

	ApprovalStatus *string `json:"approval_status" form:"approval_status"`              // pending/approved/rejected
	CreatedAtStart *string `json:"created_at_start" form:"start_time,created_at_start"` // RFC3339格式，例如：2025-01-01T00:00:00Z
	CreatedAtEnd   *string `json:"created_at_end" form:"end_time,created_at_end"`       // RFC3339格式，例如：2025-12-31T23:59:59Z
	Keyword        *string `json:"keyword" form:"keyword"`                              // 模糊搜索批次编号、发起人、发布说明
}

type BatchListParam struct {
	Page           int
	PageSize       int
	Statuses       []int8
	Initiator      *string
	ApprovalStatus *string
	CreatedAtStart *time.Time
	CreatedAtEnd   *time.Time
	Keyword        *string
}

func (q *BatchListQuery) ToParam() BatchListParam {
	param := BatchListParam{
		Page:           PageLimit(q.Page),
		PageSize:       PageSizeLimit(q.PageSize),
		Statuses:       q.Statuses,
		Initiator:      q.Initiator,
		ApprovalStatus: q.ApprovalStatus,
		Keyword:        q.Keyword,
	}

	if q.CreatedAtStart != nil && *q.CreatedAtStart != "" {
		if createTime, err := time.Parse(time.RFC3339, *q.CreatedAtStart); err == nil {
			param.CreatedAtStart = &createTime
		}
	}
	if q.CreatedAtEnd != nil && *q.CreatedAtEnd != "" {
		if createTime, err := time.Parse(time.RFC3339, *q.CreatedAtEnd); err == nil {
			param.CreatedAtEnd = &createTime
		}
	}

	return param
}

// BatchGetRequest 获取批次详情请求
type BatchGetRequest struct {
	ID               int64 `json:"id" form:"id" binding:"required"`
	AppPage          int   `json:"app_page" form:"app_page"`                     // 应用列表页码，默认1
	AppPageSize      int   `json:"app_page_size" form:"app_page_size"`           // 应用列表每页数量，默认20
	WithRecentBuilds *bool `json:"with_recent_builds" form:"with_recent_builds"` // 是否包含最近构建记录，默认true
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

// GetWithRecentBuilds 获取是否包含构建记录（默认为true）
func (q *BatchGetRequest) GetWithRecentBuilds() bool {
	if q.WithRecentBuilds == nil {
		return true
	}
	return *q.WithRecentBuilds
}

func PageLimit(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func PageSizeLimit(pageSize int) int {
	if pageSize < 1 {
		return 20
	}

	if pageSize > 100 {
		return 100
	}

	return pageSize
}
