package service

import (
	pkgErrors "devops-cd/pkg/responses"
	"fmt"
	"go.uber.org/zap"
	"time"

	"gorm.io/gorm"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/repository"
	"devops-cd/pkg/constants"
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
	SearchWithBuilds(query *dto.ApplicationSearchParam) ([]*dto.ApplicationBuildResponse, int64, error)
	GetDefaultDependencies(appID int64) (*dto.ApplicationDependenciesResponse, error)
	UpdateDefaultDependencies(appID int64, req *dto.UpdateAppDependenciesRequest) (*dto.ApplicationDependenciesResponse, error)
}

type applicationService struct {
	appRepo  *repository.ApplicationRepository
	repoRepo repository.RepositoryRepository

	db  *gorm.DB
	log *zap.SugaredLogger
}

func NewApplicationService(appRepo *repository.ApplicationRepository, repoRepo repository.RepositoryRepository, db *gorm.DB, log *zap.Logger) ApplicationService {
	return &applicationService{
		appRepo:  appRepo,
		repoRepo: repoRepo,
		db:       db,
		log:      log.With(zap.String("service", "application_service")).Sugar(),
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

	// 5. 创建环境配置
	// 如果用户指定了 EnvClusters，则使用用户配置；否则使用默认配置
	var envConfigs []model.AppEnvConfig

	if len(req.EnvClusters) > 0 {
		// 使用用户指定的环境集群配置
		for env, clusters := range req.EnvClusters {
			for _, cluster := range clusters {
				envConfigs = append(envConfigs, model.AppEnvConfig{
					AppID:      app.ID,
					Env:        env,
					Cluster:    cluster,
					Replicas:   getDefaultReplicas(env), // pre:1, prod:3, 其他:1
					BaseStatus: model.BaseStatus{Status: constants.StatusEnabled},
				})
			}
		}
	} else {
		return nil, pkgErrors.New(pkgErrors.CodeBadRequest, "未指定环境集群配置")
	}

	if len(envConfigs) > 0 {
		if err := s.db.Create(&envConfigs).Error; err != nil {
			// 回滚应用创建
			s.db.Delete(app)
			return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建环境配置失败", err)
		}
		app.EnvConfigs = envConfigs
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
	if err = s.appRepo.Update(app); err != nil {
		s.log.Errorf("更新应用失败: %v", err)
		return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "更新应用失败", err)
	}

	// 如果传入了 EnvClusters，同步更新 app_env_configs 表
	if req.EnvClusters != nil {
		if err = s.syncEnvClusters(app.ID, req.EnvClusters); err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "同步EnvClusters失败", err)
		}
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
	deps, err := s.appRepo.GetApplications(normalized)
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
		repoName := fmt.Sprintf("%s/%s", app.Repository.Namespace, app.Repository.Name)
		resp.RepoName = &repoName
	}

	// 添加团队名称
	if app.Team != nil {
		resp.TeamName = &app.Team.Name
	}

	// 添加env_clusters
	if len(app.EnvConfigs) > 0 {
		envClustersMap := make(map[string][]string)
		for _, ec := range app.EnvConfigs {
			envClustersMap[ec.Env] = append(envClustersMap[ec.Env], ec.Cluster)
		}
		resp.EnvClusters = envClustersMap
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
func (s *applicationService) SearchWithBuilds(query *dto.ApplicationSearchParam) ([]*dto.ApplicationBuildResponse, int64, error) {
	// 1. 查询应用列表（已包含最新构建信息）
	apps, total, err := s.appRepo.SearchWithBuilds(query)
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
			Description: app.Description,
			ProjectID:   app.ProjectID,
			RepoID:      app.RepoID,
			AppType:     app.AppType,
			TeamID:      app.TeamID,
			DeployedTag: app.DeployedTag,
			Status:      app.Status,

			ProjectName: app.ProjectName,
			TeamName:    app.TeamName,
		}

		// 添加代码库信息（namespace 和名称）
		if app.RepoName != nil && app.RepoNamespace != nil {
			fullname := fmt.Sprintf("%s/%s", *app.RepoNamespace, *app.RepoName)
			resp.RepoFullName = &fullname
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

// syncEnvClusters 同步更新应用的环境集群配置
// 采用增量更新策略：保留不变的配置，只删除移除的配置，只创建新增的配置
func (s *applicationService) syncEnvClusters(appID int64, envClusters map[string][]string) error {
	// 在事务中执行
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 查询现有的所有环境配置
		var existingConfigs []model.AppEnvConfig
		if err := tx.Where("app_id = ? AND deleted_at IS NULL", appID).Find(&existingConfigs).Error; err != nil {
			return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "查询现有配置失败", err)
		}

		// 2. 构建现有配置的 map，key = "env:cluster"
		existingMap := make(map[string]*model.AppEnvConfig)
		for i := range existingConfigs {
			key := fmt.Sprintf("%s:%s", existingConfigs[i].Env, existingConfigs[i].Cluster)
			existingMap[key] = &existingConfigs[i]
		}

		// 3. 处理新配置：标记哪些配置应该保留
		newConfigKeys := make(map[string]bool)
		var toCreate []model.AppEnvConfig

		for env, clusters := range envClusters {
			for _, cluster := range clusters {
				key := fmt.Sprintf("%s:%s", env, cluster)
				newConfigKeys[key] = true

				// 如果配置已存在，则保留（不做任何操作）
				if _, exists := existingMap[key]; !exists {
					// 如果不存在，则标记为需要创建
					toCreate = append(toCreate, model.AppEnvConfig{
						AppID:      appID,
						Env:        env,
						Cluster:    cluster,
						Replicas:   getDefaultReplicas(env),
						BaseStatus: model.BaseStatus{Status: constants.StatusEnabled},
					})
				}
			}
		}

		// 4. 软删除不再需要的配置
		for key, config := range existingMap {
			if !newConfigKeys[key] {
				// 这个配置不在新配置中，需要软删除
				if err := tx.Delete(config).Error; err != nil {
					return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "删除旧配置失败", err)
				}
			}
		}

		// 5. 创建新增的配置
		if len(toCreate) > 0 {
			if err := tx.Create(&toCreate).Error; err != nil {
				return pkgErrors.Wrap(pkgErrors.CodeDatabaseError, "创建新配置失败", err)
			}
		}

		return nil
	})
}

// getDefaultReplicas 根据环境返回默认副本数
func getDefaultReplicas(env string) int {
	switch env {
	case "prod":
		return 3
	default:
		return 1
	}
}
