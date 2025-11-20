package repository

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"devops-cd/internal/model"
	pkgErrors "devops-cd/pkg/errors"
)

type ApplicationRepository interface {
	Create(app *model.Application) error
	FindByID(id int64) (*model.Application, error)
	FindByName(name string) (*model.Application, error)
	FindByProjectIDAndName(projectID int64, name string) (*model.Application, error)
	FindByRepoIDAndName(repoID int64, name string) (*model.Application, error)
	List(page, pageSize int, projectID *int64, repoID *int64, teamID *int64, appType *string, keyword string, status *int8) ([]*model.Application, int64, error)
	ListByRepoID(repoID int64) ([]*model.Application, error)
	SearchWithBuilds(page, pageSize int, keyword string, projectID *int64, repoID *int64, teamID *int64, appType *string, status *int8) ([]*model.ApplicationWithBuild, int64, error)
	Update(app *model.Application) error
	Delete(id int64) error
	UpdateDefaultDependencies(appID int64, deps []int64) error
	ListAllWithDependencies() ([]*model.Application, error)
	FindByIDs(ids []int64) ([]*model.Application, error)
}

type applicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepository{db: db}
}

func (r *applicationRepository) Create(app *model.Application) error {
	if err := r.db.Create(app).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建应用失败", err)
	}
	return nil
}

func (r *applicationRepository) FindByID(id int64) (*model.Application, error) {
	var app model.Application
	err := r.db.Preload("Project").Preload("Repository").Preload("Team").First(&app, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgErrors.ErrRecordNotFound
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用失败", err)
	}
	return &app, nil
}

func (r *applicationRepository) FindByName(name string) (*model.Application, error) {
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

func (r *applicationRepository) FindByProjectIDAndName(projectID int64, name string) (*model.Application, error) {
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

func (r *applicationRepository) FindByRepoIDAndName(repoID int64, name string) (*model.Application, error) {
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

func (r *applicationRepository) List(page, pageSize int, projectID *int64, repoID *int64, teamID *int64, appType *string, keyword string, status *int8) ([]*model.Application, int64, error) {
	var apps []*model.Application
	var total int64

	query := r.db.Model(&model.Application{}).Preload("Project").Preload("Repository").Preload("Team")

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
		query = query.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
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

func (r *applicationRepository) ListByRepoID(repoID int64) ([]*model.Application, error) {
	var apps []*model.Application
	err := r.db.Where("repo_id = ? AND deleted_at IS NULL", repoID).
		Preload("Project").
		Preload("Repository").
		Preload("Team").
		Order("created_at DESC").
		Find(&apps).Error
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询代码库应用列表失败", err)
	}
	return apps, nil
}

func (r *applicationRepository) Update(app *model.Application) error {
	if err := r.db.Save(app).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新应用失败", err)
	}
	return nil
}

func (r *applicationRepository) Delete(id int64) error {
	if err := r.db.Delete(&model.Application{}, id).Error; err != nil {
		return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除应用失败", err)
	}
	return nil
}

func (r *applicationRepository) UpdateDefaultDependencies(appID int64, deps []int64) error {
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

func (r *applicationRepository) ListAllWithDependencies() ([]*model.Application, error) {
	var apps []*model.Application
	if err := r.db.Select("id", "name", "project_id", "app_type", "default_depends_on").
		Where("deleted_at IS NULL").
		Find(&apps).Error; err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用依赖信息失败", err)
	}
	return apps, nil
}

func (r *applicationRepository) FindByIDs(ids []int64) ([]*model.Application, error) {
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
func (r *applicationRepository) SearchWithBuilds(page, pageSize int, keyword string, projectID *int64, repoID *int64, teamID *int64, appType *string, status *int8) ([]*model.ApplicationWithBuild, int64, error) {
	var apps []*model.ApplicationWithBuild
	var total int64

	// 子查询：获取每个应用"上次部署后"的最新成功构建 ID
	// 逻辑：
	// 1. 如果应用有 deployed_tag，获取该 tag 对应的构建时间
	// 2. 查找该时间之后的最新成功构建
	// 3. 如果没有 deployed_tag 或之后没有新构建，返回最新成功构建
	latestBuildSubQuery := r.db.Raw(`
		SELECT 
			b1.app_id,
			b1.id as latest_build_id
		FROM builds b1
		WHERE b1.build_status = 'success'
		AND b1.id = (
			SELECT b2.id
			FROM builds b2
			LEFT JOIN applications a ON b2.app_id = a.id
			LEFT JOIN builds deployed_build ON a.deployed_tag = deployed_build.image_tag AND deployed_build.app_id = a.id
			WHERE b2.app_id = b1.app_id
			AND b2.build_status = 'success'
			AND (
				-- 情况1：有 deployed_tag，返回该时间之后的最新构建
				(a.deployed_tag IS NOT NULL 
				 AND deployed_build.id IS NOT NULL 
				 AND b2.build_created > deployed_build.build_created)
				OR
				-- 情况2：没有 deployed_tag 或 deployed_tag 之后没有新构建，返回最新构建
				(a.deployed_tag IS NULL OR deployed_build.id IS NULL)
			)
			ORDER BY b2.build_created DESC
			LIMIT 1
		)
	`)

	// 构建基础查询
	baseQuery := r.db.Model(&model.ApplicationWithBuild{}).
		Select(`
			applications.*,
			latest_builds.id as latest_build_id,
			latest_builds.build_number as latest_build_number,
			latest_builds.image_tag as latest_image_tag,
			latest_builds.commit_sha as latest_commit_sha,
			latest_builds.commit_message as latest_commit_message,
			latest_builds.commit_branch as latest_commit_branch,
			latest_builds.build_status as latest_build_status,
			latest_builds.created_at as latest_build_created_at
		`).
		Joins("LEFT JOIN repositories ON applications.repo_id = repositories.id").
		Joins("LEFT JOIN (?) AS latest_build_ids ON applications.id = latest_build_ids.app_id", latestBuildSubQuery).
		Joins("LEFT JOIN builds latest_builds ON latest_build_ids.latest_build_id = latest_builds.id").
		Preload("Project").
		Preload("Repository").
		Preload("Team").
		Where("applications.deleted_at IS NULL")

	// 过滤条件
	if projectID != nil {
		baseQuery = baseQuery.Where("applications.project_id = ?", *projectID)
	}
	if repoID != nil {
		baseQuery = baseQuery.Where("applications.repo_id = ?", *repoID)
	}
	if teamID != nil {
		baseQuery = baseQuery.Where("applications.team_id = ?", *teamID)
	}
	if appType != nil {
		baseQuery = baseQuery.Where("applications.app_type = ?", *appType)
	}
	if status != nil {
		baseQuery = baseQuery.Where("applications.status = ?", *status)
	}

	// 关键字模糊查询（如果提供）
	keywordPattern := ""
	if keyword != "" {
		keywordPattern = "%" + keyword + "%"
		baseQuery = baseQuery.Where(`(
			applications.name LIKE ? OR
			applications.display_name LIKE ? OR
			repositories.name LIKE ? OR
			latest_builds.commit_sha LIKE ? OR
			latest_builds.commit_message LIKE ? OR
			latest_builds.image_tag LIKE ?
		)`,
			keywordPattern, keywordPattern, keywordPattern,
			keywordPattern, keywordPattern, keywordPattern,
		)
	}

	// 统计总数
	// 如果有关键字，需要使用 DISTINCT 去重
	if keyword != "" {
		// 先查询去重后的应用ID列表，然后统计数量
		var distinctAppIDs []int64
		if err := baseQuery.Select("DISTINCT applications.id").Pluck("applications.id", &distinctAppIDs).Error; err != nil {
			return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计应用数量失败", err)
		}
		total = int64(len(distinctAppIDs))
	} else {
		if err := baseQuery.Count(&total).Error; err != nil {
			return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计应用数量失败", err)
		}
	}

	// 排序：按构建时间倒序（如果没有构建，按应用创建时间倒序）
	// 使用 COALESCE 处理 NULL 值
	orderBy := "COALESCE(latest_builds.created_at, applications.created_at) DESC"

	// 分页查询
	offset := (page - 1) * pageSize
	if keyword != "" {
		// 如果有关键字，需要 DISTINCT 去重
		// 先查询应用ID列表（去重且排序）
		type appIDRow struct {
			ID int64 `gorm:"column:id"`
		}

		var appIDRows []appIDRow
		if err := baseQuery.
			Select("applications.id as id, MAX(COALESCE(latest_builds.created_at, applications.created_at)) as sort_time").
			Group("applications.id").
			Order("sort_time DESC").
			Offset(offset).
			Limit(pageSize).
			Scan(&appIDRows).Error; err != nil {
			return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用ID列表失败", err)
		}

		if len(appIDRows) == 0 {
			return []*model.ApplicationWithBuild{}, total, nil
		}

		appIDs := make([]int64, 0, len(appIDRows))
		for _, row := range appIDRows {
			appIDs = append(appIDs, row.ID)
		}

		// 根据ID列表查询完整应用信息（包含构建信息）
		// 使用 FIELD 函数保持排序顺序
		orderByClause := "FIELD(applications.id"
		for _, id := range appIDs {
			orderByClause += fmt.Sprintf(", %d", id)
		}
		orderByClause += ")"

		// 重新构建完整查询（包含构建字段）
		if err := r.db.Model(&model.ApplicationWithBuild{}).
			Select(`
				applications.*,
				latest_builds.id as latest_build_id,
				latest_builds.build_number as latest_build_number,
				latest_builds.image_tag as latest_image_tag,
				latest_builds.commit_sha as latest_commit_sha,
				latest_builds.commit_message as latest_commit_message,
				latest_builds.commit_branch as latest_commit_branch,
				latest_builds.build_status as latest_build_status,
				latest_builds.created_at as latest_build_created_at
			`).
			Joins("LEFT JOIN (?) AS latest_build_ids ON applications.id = latest_build_ids.app_id", latestBuildSubQuery).
			Joins("LEFT JOIN builds latest_builds ON latest_build_ids.latest_build_id = latest_builds.id").
			Where("applications.id IN ?", appIDs).
			Preload("Repository").
			Preload("Team").
			Order(orderByClause).
			Find(&apps).Error; err != nil {
			return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用列表失败", err)
		}
	} else {
		if err := baseQuery.Order(orderBy).
			Offset(offset).
			Limit(pageSize).
			Find(&apps).Error; err != nil {
			return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询应用列表失败", err)
		}
	}

	return apps, total, nil
}
