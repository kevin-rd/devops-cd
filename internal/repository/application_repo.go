package repository

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/responses"
	"encoding/json"
	"fmt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ApplicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

func (r *ApplicationRepository) Create(app *model.Application) error {
	if err := r.db.Create(app).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建应用失败", err)
	}
	return nil
}

func (r *ApplicationRepository) FindByID(id int64) (*model.Application, error) {
	var app model.Application
	err := r.db.Preload("Project").
		Preload("Repository").
		Preload("Team").
		Preload("EnvConfigs", "deleted_at IS NULL").
		First(&app, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用失败", err)
	}
	return &app, nil
}

func (r *ApplicationRepository) FindByName(name string) (*model.Application, error) {
	var app model.Application
	err := r.db.Where("name = ? AND deleted_at IS NULL", name).First(&app).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用失败", err)
	}
	return &app, nil
}

func (r *ApplicationRepository) FindByProjectIDAndName(projectID int64, name string) (*model.Application, error) {
	var app model.Application
	err := r.db.Where("project_id = ? AND name = ? AND deleted_at IS NULL", projectID, name).First(&app).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用失败", err)
	}
	return &app, nil
}

func (r *ApplicationRepository) FindByRepoIDAndName(repoID int64, name string) (*model.Application, error) {
	var app model.Application
	err := r.db.Where("repo_id = ? AND name = ? AND deleted_at IS NULL", repoID, name).First(&app).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用失败", err)
	}
	return &app, nil
}

func (r *ApplicationRepository) List(page, pageSize int, projectID *int64, repoID *int64, teamID *int64, appType *string, keyword string, status *int8) ([]*model.Application, int64, error) {
	var apps []*model.Application
	var total int64

	query := r.db.Model(&model.Application{}).
		Preload("Project").
		Preload("Repository").
		Preload("Team").
		Preload("EnvConfigs", "deleted_at IS NULL")

	// 过滤条件
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}
	if repoID != nil {
		query = query.Where("repo_id = ?", *repoID)
	}
	if teamID != nil {
		query = query.Where("team_id = ?", *teamID)
	}
	if appType != nil {
		query = query.Where("app_type = ?", *appType)
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%")
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计应用数量失败", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用列表失败", err)
	}

	return apps, total, nil
}

func (r *ApplicationRepository) ListByRepoID(repoID int64) ([]*model.Application, error) {
	var apps []*model.Application
	err := r.db.Where("repo_id = ? AND deleted_at IS NULL", repoID).
		Preload("Project").
		Preload("Repository").
		Preload("Team").
		Preload("EnvConfigs", "deleted_at IS NULL").
		Order("created_at DESC").
		Find(&apps).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询代码库应用列表失败", err)
	}
	return apps, nil
}

func (r *ApplicationRepository) Update(app *model.Application) error {
	if err := r.db.Save(app).Error; err != nil {
		return err
	}
	return nil
}

func (r *ApplicationRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.Application{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除应用失败", err)
	}
	return nil
}

func (r *ApplicationRepository) UpdateDefaultDependencies(appID int64, deps []int64) error {
	data, err := json.Marshal(deps)
	if err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeInternalError, "序列化应用默认依赖失败", err)
	}

	if err := r.db.Model(&model.Application{}).
		Where("id = ?", appID).
		Update("default_depends_on", datatypes.JSON(data)).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新应用默认依赖失败", err)
	}

	return nil
}

func (r *ApplicationRepository) ListAllWithDependencies() ([]*model.Application, error) {
	var apps []*model.Application
	if err := r.db.Select("id", "name", "project_id", "app_type", "default_depends_on").
		Where("deleted_at IS NULL").
		Find(&apps).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用依赖信息失败", err)
	}
	return apps, nil
}

func (r *ApplicationRepository) FindByIDs(ids []int64) ([]*model.Application, error) {
	if len(ids) == 0 {
		return []*model.Application{}, nil
	}

	var apps []*model.Application
	if err := r.db.Where("id IN ? AND deleted_at IS NULL", ids).Find(&apps).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "批量查询应用失败", err)
	}
	return apps, nil
}

// SearchWithBuilds 搜索应用（包含构建信息，支持模糊查询 app、repo、commit、tag 等字段）
// 对于有 deployed_tag 的应用，返回部署后的最新构建；否则返回最新构建
// 支持多选团队和应用类型筛选
func (r *ApplicationRepository) SearchWithBuilds(param *dto.ApplicationSearchParam) ([]*model.ApplicationWithBuild, int64, error) {

	appCond := "a.deleted_at IS NULL"
	var appCondArgs []interface{}
	if param.ProjectID != nil {
		appCond += " AND a.project_id = ?"
		appCondArgs = append(appCondArgs, param.ProjectID)
	}
	if param.TeamIDs != nil && len(param.TeamIDs) > 0 {
		appCond += " AND a.team_id IN ?"
		appCondArgs = append(appCondArgs, param.TeamIDs)
	}
	if param.AppTypes != nil && len(param.AppTypes) > 0 {
		appCond += " AND a.app_type IN ?"
		appCondArgs = append(appCondArgs, param.AppTypes)
	}

	if param.Keyword != "" {
		appCond += " AND a.name LIKE ?"
		appCondArgs = append(appCondArgs, "%"+param.Keyword+"%")
	}

	// COUNT 查询
	var total int64
	if err := r.db.Table("applications a").Where(appCond, appCondArgs...).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 主查询, 分页
	offset := (param.Page - 1) * param.PageSize
	sql := fmt.Sprintf(`
WITH deployed AS (
	SELECT 
		a.id AS app_id,
		db.build_created AS deployed_time
	FROM applications a
	LEFT JOIN builds db ON a.deployed_tag = db.image_tag AND db.app_id = a.id
	WHERE %s
),
candidates AS (
	SELECT 
		b.*,
		d.deployed_time,
		ROW_NUMBER() OVER (PARTITION BY b.app_id ORDER BY b.build_created DESC) AS rn_created
	FROM builds b
	LEFT JOIN deployed d ON b.app_id = d.app_id
	WHERE b.build_status = 'success' AND (d.deployed_time IS NULL OR b.build_created > d.deployed_time)
)
SELECT 
	a.*,
	p.name AS project_name,
	t.name AS team_name,
	r.namespace AS repo_namespace,
	r.name AS repo_name,
	c.id AS latest_build_id,
	c.build_number AS latest_build_number,
	c.image_tag AS latest_image_tag,
	c.commit_sha AS latest_commit_sha,
	c.commit_message AS latest_commit_message,
	c.commit_branch AS latest_commit_branch,
	c.build_status AS latest_build_status,
	c.build_created AS latest_build_created_at
FROM applications a
LEFT JOIN candidates c ON c.app_id = a.id AND c.rn_created = 1
LEFT JOIN projects p ON a.project_id = p.id
LEFT JOIN teams t ON a.team_id = t.id
LEFT JOIN repositories r ON a.repo_id = r.id
WHERE %s
ORDER BY c.build_created DESC
LIMIT ? OFFSET ?
	`, appCond, appCond)

	var apps []*model.ApplicationWithBuild
	if err := r.db.Raw(sql, append(appCondArgs, append(appCondArgs, param.PageSize, offset)...)...).Scan(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}
