package repository

import (
	pkgErrors "devops-cd/pkg/responses"
	"gorm.io/gorm"

	"devops-cd/internal/model"
)

type ProjectRepository interface {
	Create(project *model.Project) error
	FindByID(id int64) (*model.Project, error)
	FindByName(name string) (*model.Project, error)
	List(page, pageSize int, keyword string) ([]*model.Project, int64, error)
	ListAll() ([]*model.Project, error)
	Update(project *model.Project) error
	Delete(id int64) error
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(project *model.Project) error {
	if err := r.db.Create(project).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建项目失败", err)
	}
	return nil
}

func (r *projectRepository) FindByID(id int64) (*model.Project, error) {
	var project model.Project
	err := r.db.First(&project, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目失败", err)
	}
	return &project, nil
}

func (r *projectRepository) FindByName(name string) (*model.Project, error) {
	var project model.Project
	err := r.db.Where("name = ? AND deleted_at IS NULL", name).First(&project).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目失败", err)
	}
	return &project, nil
}

func (r *projectRepository) List(page, pageSize int, keyword string) ([]*model.Project, int64, error) {
	var projects []*model.Project
	var total int64

	query := r.db.Model(&model.Project{})

	// 关键字搜索
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计项目数量失败", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目列表失败", err)
	}

	return projects, total, nil
}

func (r *projectRepository) ListAll() ([]*model.Project, error) {
	var projects []*model.Project
	err := r.db.Where("deleted_at IS NULL").Order("name ASC").Find(&projects).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询项目列表失败", err)
	}
	return projects, nil
}

func (r *projectRepository) Update(project *model.Project) error {
	if err := r.db.Save(project).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新项目失败", err)
	}
	return nil
}

func (r *projectRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.Project{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除项目失败", err)
	}
	return nil
}
