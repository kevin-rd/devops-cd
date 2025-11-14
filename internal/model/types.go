package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Int64List []int64

// Scan 实现 sql.Scanner
func (l *Int64List) Scan(value interface{}) error {
	if value == nil {
		*l = []int64{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, l)
	case string: // MySQL、SQL Server 等可能返回 string
		return json.Unmarshal([]byte(v), l)
	default:
		return fmt.Errorf("cannot scan %T into Int64List", value)
	}
}

// Value 实现 driver.Valuer
func (l Int64List) Value() (driver.Value, error) {
	if len(l) == 0 {
		// 必须返回 []byte，不要返回 string
		// gorm + mysql 对 JSON 类型处理更稳定
		return []byte("[]"), nil
	}
	return json.Marshal(l)
}
