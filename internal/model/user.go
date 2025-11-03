package model

import "time"

// User 本地用户模型
type User struct {
	BaseStatus
	Username    string     `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Password    string     `gorm:"size:255;not null" json:"-"` // 不返回到前端
	Email       *string    `gorm:"size:100" json:"email"`
	DisplayName *string    `gorm:"size:100" json:"display_name"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// Team 团队模型
type Team struct {
	BaseStatus
	Name        string  `gorm:"size:100;not null;uniqueIndex" json:"name"`
	DisplayName *string `gorm:"size:100" json:"display_name"`
	Description *string `gorm:"type:text" json:"description"`
	LeaderName  *string `gorm:"size:100" json:"leader_name"`
}

func (Team) TableName() string {
	return "teams"
}
