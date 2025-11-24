package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/repository"
	"devops-cd/pkg/constants"
	pkgErrors "devops-cd/pkg/errors"
)

type ApplicationService interface {
	Create(req *dto.CreateApplicationRequest) (*dto.ApplicationResponse, error)
	GetByID(id int64) (*dto.ApplicationResponse, error)
	List(query *dto.ApplicationListQuery) ([]*dto.ApplicationResponse, int64, error)
	Update(id int64, req *dto.UpdateApplicationRequest) (*dto.ApplicationResponse, error)
	Delete(id int64) error
	GetBuilds(id int64, page, pageSize int) ([]*dto.ApplicationBuildInfo, int64, error)
	ListByRepoID(repoID int64) ([]*dto.ApplicationResponse, error)
	GetAppTypes() (*dto.AppTypesResponse, error)
	SearchWithBuilds(query *dto.ApplicationSearchQuery) ([]*dto.ApplicationBuildResponse, int64, error)
	GetDefaultDependencies(appID int64) (*dto.ApplicationDependenciesResponse, error)
	UpdateDefaultDependencies(appID int64, req *dto.UpdateAppDependenciesRequest) (*dto.ApplicationDependenciesResponse, error)
}

type applicationService struct {
	appRepo  repository.ApplicationRepository
	repoRepo repository.RepositoryRepository
	db       *gorm.DB
}

func NewApplicationService(appRepo repository.ApplicationRepository, repoRepo repository.RepositoryRepository, db *gorm.DB) ApplicationService {
	return &applicationService{
		appRepo:  appRepo,
		repoRepo: repoRepo,
		db:       db,
	}
}

func (s *applicationService) Create(req *dto.CreateApplicationRequest) (*dto.ApplicationResponse, error) {
	// 1. 获取代码库信息
	repo, err := s.repoRepo.FindByID(req.RepoID)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "关联的代码库不存在", nil)
		}
		return nil, err
	}

	// 2. 确定使用的 project_id（优先使用请求中指定的，否则从 repository 继承）
	var projectID int64
	if req.ProjectID != nil {
		projectID = *req.ProjectID
	} else if repo.ProjectID != nil {
		projectID = *repo.ProjectID
	} else {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "必须指定项目或代码库已关联项目", nil)
	}

	// 3. 检查应用名称在同一项目下是否已存在
	existing, _ := s.appRepo.FindByProjectIDAndName(projectID, req.Name)
	if existing != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("应用 %s 在该项目中已存在", req.Name), nil)
	}

	// 4. 创建应用
	app := &model.Application{
		Name:        req.Name,
		ProjectID:   projectID,
		DisplayName: req.DisplayName,
		Description: req.Description,
		RepoID:      req.RepoID,
		AppType:     req.AppType,
		TeamID:      req.TeamID,
		BaseStatus: model.BaseStatus{
			Status: constants.StatusEnabled,
		},
	}

	if err := s.appRepo.Create(app); err != nil {
		return nil, err
	}

	// 5. 创建默认环境配置
	defaultConfigs := []model.AppEnvConfig{
		{
			AppID:      app.ID,
			Env:        "pre",
			Cluster:    "default",
			Replicas:   1,
			BaseStatus: model.BaseStatus{Status: constants.StatusEnabled},
		},
		{
			AppID:      app.ID,
			Env:        "prod",
			Cluster:    "default",
			Replicas:   3,
			BaseStatus: model.BaseStatus{Status: constants.StatusEnabled},
		},
	}
	if err := s.db.Create(&defaultConfigs).Error; err != nil {
		// 回滚应用创建
		s.db.Delete(app)
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建默认环境配置失败", err)
	}

	// 手动设置关联数据以便正确返回
	app.Repository = repo

	// 如果有 ProjectID，尝试加载 Project 信息
	var project model.Project
	if err := s.db.First(&project, projectID).Error; err == nil {
		app.Project = &project
	}

	// 如果有 TeamID，尝试加载 Team 信息
	if req.TeamID != nil {
		var team model.Team
		if err := s.db.First(&team, *req.TeamID).Error; err == nil {
			app.Team = &team
		}
		// 如果加载失败，忽略错误，team_name只是为了方便前端显示
	}

	return s.toResponse(app), nil
}

func (s *applicationService) GetByID(id int64) (*dto.ApplicationResponse, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(app), nil
}

func (s *applicationService) List(query *dto.ApplicationListQuery) ([]*dto.ApplicationResponse, int64, error) {
	apps, total, err := s.appRepo.List(
		query.GetPage(),
		query.GetPageSize(),
		query.ProjectID,
		query.RepoID,
		query.TeamID,
		query.AppType,
		query.Keyword,
		query.Status,
	)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.ApplicationResponse, len(apps))
	for i, app := range apps {
		responses[i] = s.toResponse(app)
	}

	return responses, total, nil
}

func (s *applicationService) Update(id int64, req *dto.UpdateApplicationRequest) (*dto.ApplicationResponse, error) {
	// 查询应用
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 检查名称是否冲突（在同一项目下唯一）
	if req.Name != nil && *req.Name != app.Name {
		existing, _ := s.appRepo.FindByProjectIDAndName(app.ProjectID, *req.Name)
		if existing != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("应用 %s 在该项目中已存在", *req.Name), nil)
		}
		app.Name = *req.Name
	}

	// 注意：不允许修改 repo_id 和 project_id，保证数据一致性

	// 更新字段
	if req.DisplayName != nil {
		app.DisplayName = req.DisplayName
	}
	if req.Description != nil {
		app.Description = req.Description
	}
	if req.AppType != nil {
		app.AppType = *req.AppType
	}
	if req.TeamID != nil {
		app.TeamID = req.TeamID
	}
	if req.DeployedTag != nil {
		app.DeployedTag = req.DeployedTag
	}
	if req.Status != nil {
		app.Status = *req.Status
	}

	// 保存更新
	if err := s.appRepo.Update(app); err != nil {
		return nil, err
	}

	// 重新查询以获取关联数据
	app, err = s.appRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(app), nil
}

func (s *applicationService) Delete(id int64) error {
	// 检查应用是否存在
	_, err := s.appRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 软删除应用（不级联删除Build记录）
	return s.appRepo.Delete(id)
}

func (s *applicationService) GetBuilds(id int64, page, pageSize int) ([]*dto.ApplicationBuildInfo, int64, error) {
	// 检查应用是否存在
	_, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, 0, err
	}

	// 查询构建记录
	var builds []*model.Build
	var total int64

	query := s.db.Model(&model.Build{}).Where("app_id = ?", id)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "统计构建数量失败", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&builds).Error; err != nil {
		return nil, 0, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询构建列表失败", err)
	}

	// 转换为DTO
	buildInfos := make([]*dto.ApplicationBuildInfo, len(builds))
	for i, build := range builds {
		buildInfos[i] = s.toBuildInfo(build)
	}

	return buildInfos, total, nil
}

func (s *applicationService) ListByRepoID(repoID int64) ([]*dto.ApplicationResponse, error) {
	// 直接查询应用列表，不验证代码库是否存在
	// 如果代码库不存在或被软删除，返回空列表更合理
	apps, err := s.appRepo.ListByRepoID(repoID)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.ApplicationResponse, len(apps))
	for i, app := range apps {
		responses[i] = s.toResponse(app)
	}

	return responses, nil
}

func (s *applicationService) GetDefaultDependencies(appID int64) (*dto.ApplicationDependenciesResponse, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	return s.buildDependenciesResponse(app)
}

func (s *applicationService) UpdateDefaultDependencies(appID int64, req *dto.UpdateAppDependenciesRequest) (*dto.ApplicationDependenciesResponse, error) {
	if _, err := s.appRepo.FindByID(appID); err != nil {
		return nil, err
	}

	normalized := normalizeDependencyIDs(req.Dependencies)
	for _, depID := range normalized {
		if depID == appID {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "应用不能依赖自身", nil)
		}
	}

	// 校验依赖是否存在
	deps, err := s.appRepo.FindByIDs(normalized)
	if err != nil {
		return nil, err
	}
	if len(deps) != len(normalized) {
		existing := make(map[int64]struct{}, len(deps))
		for _, dep := range deps {
			existing[dep.ID] = struct{}{}
		}
		missing := make([]int64, 0)
		for _, depID := range normalized {
			if _, ok := existing[depID]; !ok {
				missing = append(missing, depID)
			}
		}
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("存在不存在的依赖应用: %v", missing), nil)
	}

	// 构建依赖图做循环检测
	apps, err := s.appRepo.ListAllWithDependencies()
	if err != nil {
		return nil, err
	}

	graph := make(map[int64][]int64, len(apps))
	for _, item := range apps {
		ids := item.DefaultDependsOn

		if item.ID == appID {
			ids = normalized
		}
		graph[item.ID] = ids
	}
	if _, exists := graph[appID]; !exists {
		graph[appID] = normalized
	}

	if hasDependencyCycle(graph) {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "依赖配置存在循环，请调整", nil)
	}

	if err := s.appRepo.UpdateDefaultDependencies(appID, normalized); err != nil {
		return nil, err
	}

	updated, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	return s.buildDependenciesResponse(updated)
}

// toResponse 转换为响应对象
func (s *applicationService) toResponse(app *model.Application) *dto.ApplicationResponse {
	resp := &dto.ApplicationResponse{
		ID:          app.ID,
		Name:        app.Name,
		DisplayName: app.DisplayName,
		Description: app.Description,
		ProjectID:   app.ProjectID,
		RepoID:      app.RepoID,
		AppType:     app.AppType,
		TeamID:      app.TeamID,
		DeployedTag: app.DeployedTag,
		Status:      app.Status,
		CreatedAt:   app.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   app.UpdatedAt.Format(time.RFC3339),
	}

	resp.DefaultDependsOn = app.DefaultDependsOn

	// 添加项目名称
	if app.Project != nil {
		resp.ProjectName = &app.Project.Name
	}

	// 添加代码库信息（namespace 和名称）
	if app.Repository != nil {
		resp.Namespace = app.Repository.Namespace // 从 Repository 获取 namespace
		repoName := fmt.Sprintf("%s/%s", app.Repository.Namespace, app.Repository.Name)
		resp.RepoName = &repoName
	}

	// 添加团队名称
	if app.Team != nil {
		resp.TeamName = &app.Team.Name
	}

	return resp
}

// toBuildInfo 转换构建信息
func (s *applicationService) toBuildInfo(build *model.Build) *dto.ApplicationBuildInfo {
	commitMsg := &build.CommitMessage
	duration := build.BuildDuration

	info := &dto.ApplicationBuildInfo{
		ID:            build.ID,
		BuildNumber:   fmt.Sprintf("%d", build.BuildNumber),
		Tag:           build.ImageTag,
		Branch:        build.CommitBranch,
		CommitID:      build.CommitSHA,
		CommitMessage: commitMsg,
		BuildStatus:   build.BuildStatus,
		TriggerType:   build.BuildEvent,
		Duration:      &duration,
		CreatedAt:     build.CreatedAt.Format(time.RFC3339),
		StartedAt:     nil, // 待赋值
		FinishedAt:    nil, // 待赋值
	}

	// BuildStarted 和 BuildFinished 现在是 time.Time，直接格式化
	startedAt := build.BuildStarted.Format(time.RFC3339)
	info.StartedAt = &startedAt

	finishedAt := build.BuildFinished.Format(time.RFC3339)
	info.FinishedAt = &finishedAt

	return info
}

// GetAppTypes 获取应用类型列表
func (s *applicationService) GetAppTypes() (*dto.AppTypesResponse, error) {
	metadata := config.GetAppTypeMetadata()

	types := make([]dto.AppTypeInfo, 0, len(metadata))
	for _, meta := range metadata {
		desc := meta.Description
		icon := meta.Icon
		color := meta.Color

		types = append(types, dto.AppTypeInfo{
			Value:       meta.Value,
			Label:       meta.Label,
			Description: &desc,
			Icon:        &icon,
			Color:       &color,
		})
	}

	return &dto.AppTypesResponse{
		Types: types,
		Total: len(types),
	}, nil
}

// SearchWithBuilds 搜索应用（包含构建信息，支持模糊查询）
func (s *applicationService) SearchWithBuilds(query *dto.ApplicationSearchQuery) ([]*dto.ApplicationBuildResponse, int64, error) {
	// 1. 查询应用列表（已包含最新构建信息）
	apps, total, err := s.appRepo.SearchWithBuilds(
		query.GetPage(),
		query.GetPageSize(),
		query.Keyword,
		query.ProjectID,
		query.RepoID,
		query.TeamIDs,  // 支持多选
		query.AppTypes, // 支持多选
		query.Status,
	)
	if err != nil {
		return nil, 0, err
	}

	if len(apps) == 0 {
		return []*dto.ApplicationBuildResponse{}, total, nil
	}

	// 2. 转换为响应格式
	responses := make([]*dto.ApplicationBuildResponse, len(apps))
	for i, app := range apps {
		resp := &dto.ApplicationBuildResponse{
			ID:          app.ID,
			Name:        app.Name,
			DisplayName: app.DisplayName,
			Description: app.Description,
			ProjectID:   app.ProjectID,
			RepoID:      app.RepoID,
			AppType:     app.AppType,
			TeamID:      app.TeamID,
			DeployedTag: app.DeployedTag,
			Status:      app.Status,
		}

		// 添加项目名称
		if app.Project != nil {
			resp.ProjectName = &app.Project.Name
		}

		// 添加代码库信息（namespace 和名称）
		if app.Repository != nil {
			resp.Namespace = app.Repository.Namespace // 从 Repository 获取 namespace
			repoFullName := fmt.Sprintf("%s/%s", app.Repository.Namespace, app.Repository.Name)
			resp.RepoFullName = &repoFullName
		}

		// 添加团队名称
		if app.Team != nil {
			resp.TeamName = &app.Team.Name
		}

		// 添加最新构建信息
		if app.LatestBuildID != nil {
			resp.BuildID = *app.LatestBuildID
			resp.BuildNumber = *app.LatestBuildNumber
			resp.BuildTime = app.LatestBuildCreatedAt
			resp.ImageTag = *app.LatestImageTag
			resp.CommitSHA = *app.LatestCommitSHA
			resp.CommitMessage = app.LatestCommitMessage
			resp.CommitBranch = *app.LatestCommitBranch
			resp.BuildStatus = *app.LatestBuildStatus
		}

		responses[i] = resp
	}

	return responses, total, nil
}

func (s *applicationService) buildDependenciesResponse(app *model.Application) (*dto.ApplicationDependenciesResponse, error) {
	deps := app.DefaultDependsOn

	return &dto.ApplicationDependenciesResponse{
		AppID:        app.ID,
		AppName:      app.Name,
		Dependencies: deps,
		UpdatedAt:    app.UpdatedAt.Format(time.RFC3339),
	}, nil
}
