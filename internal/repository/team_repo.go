package repository

import (
	"gorm.io/gorm"

	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/errors"
)

type TeamRepository interface {
	Create(team *model.Team) error
	FindByID(id int64) (*model.Team, error)
	FindByName(name string) (*model.Team, error)
	ListAll() ([]*model.Team, error)
	ListByProjectID(projectID int64) ([]*model.Team, error)
	ListByProjectIDs(projectIDs []int64) ([]*model.Team, error)
	Update(team *model.Team) error
	Delete(id int64) error
	// VerifyTeamBelongsToProject 验证团队是否属于指定项目
	VerifyTeamBelongsToProject(teamID int64, projectID int64) (bool, error)
}

type teamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) TeamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) Create(team *model.Team) error {
	if err := r.db.Create(team).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建团队失败", err)
	}
	return nil
}

func (r *teamRepository) FindByID(id int64) (*model.Team, error) {
	var team model.Team
	err := r.db.First(&team, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队失败", err)
	}
	return &team, nil
}

func (r *teamRepository) FindByName(name string) (*model.Team, error) {
	var team model.Team
	err := r.db.Where("name = ? AND deleted_at IS NULL", name).First(&team).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队失败", err)
	}
	return &team, nil
}

func (r *teamRepository) ListAll() ([]*model.Team, error) {
	var teams []*model.Team
	err := r.db.Where("deleted_at IS NULL").Order("project_id ASC, name ASC").Find(&teams).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队列表失败", err)
	}
	return teams, nil
}

func (r *teamRepository) ListByProjectID(projectID int64) ([]*model.Team, error) {
	var teams []*model.Team
	err := r.db.Where("project_id = ? AND deleted_at IS NULL", projectID).
		Order("name ASC").Find(&teams).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队列表失败", err)
	}
	return teams, nil
}

func (r *teamRepository) ListByProjectIDs(projectIDs []int64) ([]*model.Team, error) {
	if len(projectIDs) == 0 {
		return []*model.Team{}, nil
	}
	var teams []*model.Team
	err := r.db.Where("project_id IN ? AND deleted_at IS NULL", projectIDs).
		Order("project_id ASC, name ASC").Find(&teams).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询团队列表失败", err)
	}
	return teams, nil
}

func (r *teamRepository) Update(team *model.Team) error {
	if err := r.db.Save(team).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新团队失败", err)
	}
	return nil
}

func (r *teamRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.Team{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除团队失败", err)
	}
	return nil
}

// VerifyTeamBelongsToProject 验证团队是否属于指定项目
func (r *teamRepository) VerifyTeamBelongsToProject(teamID int64, projectID int64) (bool, error) {
	var count int64
	err := r.db.Model(&model.Team{}).
		Where("id = ? AND project_id = ? AND deleted_at IS NULL", teamID, projectID).
		Count(&count).Error
	if err != nil {
		return false, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "验证团队归属失败", err)
	}
	return count > 0, nil
}
