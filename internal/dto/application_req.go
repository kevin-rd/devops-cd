package dto

// ApplicationSearchQuery 应用搜索查询参数（包含构建信息）
type ApplicationSearchQuery struct {
	PageQuery // 分页参数（page, page_size, keyword）

	ProjectID *int64   `form:"project_id"` // 可选：按项目ID过滤
	RepoID    *int64   `form:"repo_id"`    // 可选：按代码库ID过滤
	TeamIDs   []int64  `form:"team_ids"`   // 可选：按团队ID过滤（多选）
	AppTypes  []string `form:"app_types"`  // 可选：按应用类型过滤（多选）
}

type ApplicationSearchParam struct {
	Page     int
	PageSize int

	Keyword string

	ProjectID *int64
	TeamIDs   []int64
	AppTypes  []string
}

func (q *ApplicationSearchQuery) ToParam() ApplicationSearchParam {
	return ApplicationSearchParam{
		Page:     q.GetPage(),
		PageSize: q.GetPageSize(),

		Keyword: q.Keyword,

		ProjectID: q.ProjectID,
		TeamIDs:   q.TeamIDs,
		AppTypes:  q.AppTypes,
	}
}
