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
	ProjectID   int64   `gorm:"not null;index" json:"project_id"`
	Description *string `gorm:"type:text" json:"description"`
	LeaderName  *string `gorm:"size:100" json:"leader_name"`
}

func (Team) TableName() string {
	return "teams"
}

// Project 项目模型
type Project struct {
	BaseModelWithSoftDelete
	Name        string  `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description *string `gorm:"type:text" json:"description"`
	OwnerName   *string `gorm:"size:100" json:"owner_name"`
}

func (Project) TableName() string {
	return "projects"
}

// TeamMember 团队成员
type TeamMember struct {
	BaseStatus
	TeamID int64  `gorm:"column:team_id;not null;index" json:"team_id"`
	UserID int64  `gorm:"column:user_id;not null;index" json:"user_id"`
	Role   string `gorm:"size:20;not null;default:'member'" json:"role"`

	Team *Team `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (TeamMember) TableName() string {
	return "team_members"
}
