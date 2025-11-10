package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Int64List []int64

// 实现 sql.Scanner
func (l *Int64List) Scan(value interface{}) error {
	if value == nil {
		*l = []int64{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into Int64List", value)
	}
	return json.Unmarshal(bytes, l)
}

// 实现 driver.Valuer
func (l Int64List) Value() (driver.Value, error) {
	if len(l) == 0 {
		return "[]", nil
	}
	return json.Marshal(l)
}
