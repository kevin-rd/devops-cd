package transitions

import (
	"devops-cd/pkg/constants"
	"gorm.io/gorm"
)

// StatusIn 批量查询指定范围内的状态, 左闭右开区间
func StatusIn(status int8) func(db *gorm.DB) *gorm.DB {
	start, end := constants.Range10(status)
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status >= ? AND status < ?", start, end)
	}
}
