package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
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

	// 2. 检查应用名称在同一project下是否已存在
	existing, _ := s.appRepo.FindByProjectAndName(repo.Project, req.Name)
	if existing != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("应用 %s 在项目 %s 中已存在", req.Name, repo.Project), nil)
	}

	// 3. 创建应用（自动继承project）
	app := &model.Application{
		Name:        req.Name,
		Project:     repo.Project, // 继承自repository
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

	// 手动设置关联数据以便正确返回
	app.Repository = repo

	// 如果有TeamID，尝试加载Team信息
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

	// 检查名称是否冲突（在同一project下唯一）
	if req.Name != nil && *req.Name != app.Name {
		existing, _ := s.appRepo.FindByProjectAndName(app.Project, *req.Name)
		if existing != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("应用 %s 在项目 %s 中已存在", *req.Name, app.Project), nil)
		}
		app.Name = *req.Name
	}

	// 注意：不允许修改repo_id和project，保证数据一致性

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

// toResponse 转换为响应对象
func (s *applicationService) toResponse(app *model.Application) *dto.ApplicationResponse {
	resp := &dto.ApplicationResponse{
		ID:          app.ID,
		Name:        app.Name,
		Project:     app.Project,
		DisplayName: app.DisplayName,
		Description: app.Description,
		RepoID:      app.RepoID,
		AppType:     app.AppType,
		TeamID:      app.TeamID,
		DeployedTag: app.DeployedTag,
		Status:      app.Status,
		CreatedAt:   app.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   app.UpdatedAt.Format(time.RFC3339),
	}

	// 添加代码库名称
	if app.Repository != nil {
		repoName := fmt.Sprintf("%s/%s", app.Repository.Project, app.Repository.Name)
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
	}

	if build.BuildStarted > 0 {
		startedAt := time.Unix(build.BuildStarted, 0).Format(time.RFC3339)
		info.StartedAt = &startedAt
	}

	if build.BuildFinished > 0 {
		finishedAt := time.Unix(build.BuildFinished, 0).Format(time.RFC3339)
		info.FinishedAt = &finishedAt
	}

	return info
}

// GetAppTypes 获取应用类型列表
func (s *applicationService) GetAppTypes() (*dto.AppTypesResponse, error) {
	metadata := constants.GetAppTypeMetadata()

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
		query.RepoID,
		query.TeamID,
		query.AppType,
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
			Project:     app.Project,
			DisplayName: app.DisplayName,
			Description: app.Description,
			RepoID:      app.RepoID,
			AppType:     app.AppType,
			TeamID:      app.TeamID,
			DeployedTag: app.DeployedTag,
			Status:      app.Status,
		}

		// 添加代码库名称
		if app.Repository != nil {
			repoName := fmt.Sprintf("%s/%s", app.Repository.Project, app.Repository.Name)
			resp.RepoName = &repoName
		}

		// 添加团队名称
		if app.Team != nil {
			resp.TeamName = &app.Team.Name
		}

		// 添加最新构建信息
		if app.LatestBuildID != nil {
			resp.BuildID = *app.LatestBuildID
			resp.BuildNumber = *app.LatestBuildNumber
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
