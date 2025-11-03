package dto

// AppTypeInfo 应用类型信息
type AppTypeInfo struct {
	Value       string  `json:"value"`       // 类型值（用于API传参）
	Label       string  `json:"label"`       // 显示名称
	Description *string `json:"description"` // 描述（可选）
	Icon        *string `json:"icon"`        // 图标（可选）
	Color       *string `json:"color"`       // 颜色标识（可选）
}

// AppTypesResponse 应用类型列表响应
type AppTypesResponse struct {
	Types []AppTypeInfo `json:"types"` // 应用类型列表
	Total int           `json:"total"` // 类型总数
}
