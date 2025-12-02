package model

import (
	"database/sql"
)

type Scope string
type ValueType string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"

	TypeString ValueType = "string"
	TypeNumber ValueType = "number"
	TypeJSON   ValueType = "json"
	TypeSecret ValueType = "secret"
)

type ConfigItem struct {
	BaseModel

	Scope     Scope         `gorm:"not null" json:"scope"`
	ProjectID sql.NullInt64 `gorm:"not null" json:"project_id"`
	Key       string        `gorm:"column:config_key;not null" json:"key"`
	Value     string        `gorm:"column:config_value;not null" json:"value"`
	ValueType ValueType     `gorm:"not null" json:"value_type"`
}
