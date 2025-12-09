package repository

import (
	pkgErrors "devops-cd/pkg/responses"
	"errors"
	"gorm.io/gorm"

	"devops-cd/internal/model"
)

type TeamMemberRepository struct {
	db *gorm.DB
}

func NewTeamMemberRepository(db *gorm.DB) *TeamMemberRepository {
	return &TeamMemberRepository{db: db}
}

func (r *TeamMemberRepository) Create(member *model.TeamMember) error {
	if err := r.db.Create(member).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "添加团队成员失败", err)
	}
	return nil
}

func (r *TeamMemberRepository) Update(member *model.TeamMember) error {
	if err := r.db.Save(member).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新团队成员失败", err)
	}
	return nil
}

func (r *TeamMemberRepository) FindByID(id int64) (*model.TeamMember, error) {
	var member model.TeamMember
	err := r.db.Preload("User").First(&member, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队成员失败", err)
	}
	return &member, nil
}

func (r *TeamMemberRepository) FindByTeamAndUser(teamID, userID int64) (*model.TeamMember, error) {
	var member model.TeamMember
	err := r.db.Where("team_id = ? AND user_id = ?", teamID, userID).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队成员失败", err)
	}
	return &member, nil
}

func (r *TeamMemberRepository) FindByProjectAndUser(projectID, userID int64) (*model.TeamMember, error) {
	var member model.TeamMember
	err := r.db.Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队成员失败", err)
	}
	return &member, nil
}

func (r *TeamMemberRepository) ListByTeam(teamID int64, page, pageSize int, keyword string) ([]*model.TeamMember, int64, error) {
	var members []*model.TeamMember
	var total int64

	query := r.db.Model(&model.TeamMember{}).
		Where("team_id = ?", teamID).
		Preload("User")

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Joins("LEFT JOIN users ON users.id = team_members.user_id").
			Where("users.username LIKE ? OR users.display_name LIKE ?", like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计团队成员失败", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("team_members.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&members).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队成员失败", err)
	}

	return members, total, nil
}

func (r *TeamMemberRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.TeamMember{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除团队成员失败", err)
	}
	return nil
}
