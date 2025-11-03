package dto

// PageQuery 分页查询参数
type PageQuery struct {
	Page     int    `form:"page"`                                 // 可选：页码，不传默认为1
	PageSize int    `form:"page_size"`                            // 可选：每页数量，不传默认为10
	Keyword  string `form:"keyword"`                              // 可选：关键字搜索
	Status   *int8  `form:"status" binding:"omitempty,oneof=0 1"` // 可选：状态过滤
}

// GetPage 获取页码
func (p *PageQuery) GetPage() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

// GetPageSize 获取每页数量
func (p *PageQuery) GetPageSize() int {
	if p.PageSize < 1 {
		return 10
	}
	if p.PageSize > 100 {
		return 100
	}
	return p.PageSize
}

// GetOffset 获取偏移量
func (p *PageQuery) GetOffset() int {
	return (p.GetPage() - 1) * p.GetPageSize()
}

// IDParam ID参数
type IDParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// PageResponse 分页响应
type PageResponse struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// NewPageResponse 创建分页响应
func NewPageResponse(items interface{}, total int64, page, pageSize int) *PageResponse {
	return &PageResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}
