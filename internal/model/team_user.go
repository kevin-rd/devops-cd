package model

import "time"

const UserTableName = "users"
const TeamTableName = "teams"
const TeamMemberTableName = "team_members"

// User 本地用户模型
type User struct {
	BaseStatus
	AuthProvider string     `gorm:"size:20;not null;default:local;uniqueIndex:idx_user_provider_name" json:"auth_provider"`
	Username     string     `gorm:"size:50;not null;uniqueIndex:idx_user_provider_name" json:"username"`
	Password     string     `gorm:"size:255" json:"-"` // 不返回到前端；LDAP 用户可为空字符串
	ExternalUID  *string    `gorm:"size:191" json:"external_uid,omitempty"`
	Email        *string    `gorm:"size:100" json:"email,omitempty"`
	DisplayName  *string    `gorm:"size:100" json:"display_name,omitempty"`
	Phone        *string    `gorm:"size:32" json:"phone,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	SystemRoles  StringList `gorm:"column:system_roles;type:json" json:"system_roles"`

	TeamMembers []TeamMember `gorm:"foreignKey:UserID;references:ID" json:"team_members,omitempty"`
	//Teams       []Team       `gorm:"foreignKey:many2many:team_members;joinForeignKey:userid,joinReferences:team_id" json:"teams,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return UserTableName
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
	return TeamTableName
}

// TeamMember 团队成员
type TeamMember struct {
	BaseModel

	TeamID int64      `gorm:"column:team_id;not null;index" json:"team_id"`
	UserID int64      `gorm:"column:user_id;not null;index" json:"user_id"`
	Roles  StringList `gorm:"column:roles;type:json" json:"roles"`

	// Relations
	Team *Team `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (TeamMember) TableName() string {
	return TeamMemberTableName
}
