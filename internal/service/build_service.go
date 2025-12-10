package service

import (
	"devops-cd/internal/core"
	pkgErrors "devops-cd/pkg/responses"
	"devops-cd/pkg/utils"
	"fmt"
	"strings"
	"time"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/repository"
	"go.uber.org/zap"
)

// BuildService 构建服务接口
type BuildService interface {
	ProcessNotify(req *dto.BuildNotifyRequest) error
	GetByID(id int64) (*dto.BuildResponse, error)
	GetByAppAndNumber(appID int64, buildNumber int) (*dto.BuildResponse, error)
	List(query *dto.BuildListQuery) ([]*dto.BuildResponse, int64, error)
	ListByRepoID(repoID int64, limit int) ([]*dto.BuildResponse, error)
	ListByAppID(appID int64, limit int) ([]*dto.BuildResponse, error)
}

type buildService struct {
	buildRepo  repository.BuildRepository
	repoRepo   repository.RepositoryRepository
	appRepo    *repository.ApplicationRepository
	coreEngine *core.CoreEngine
}

// NewBuildService 创建构建服务实例
func NewBuildService(buildRepo repository.BuildRepository, repoRepo repository.RepositoryRepository, appRepo *repository.ApplicationRepository, coreEngine *core.CoreEngine) BuildService {
	return &buildService{
		buildRepo:  buildRepo,
		repoRepo:   repoRepo,
		appRepo:    appRepo,
		coreEngine: coreEngine,
	}
}

// ProcessNotify 处理构建通知（Drone webhook）
func (s *buildService) ProcessNotify(req *dto.BuildNotifyRequest) error {
	log := logger.Log.With(zap.String("handler", "BuildService.ProcessNotify"), zap.String("repo", req.Repo)).Sugar()
	log.Infof("收到构建通知: %s :%v: %s", req.Repo, req.BuildNumber, req.BuildStatus)

	// 1. 解析仓库信息
	project, name := parseRepo(req.Repo)
	if project == "" || name == "" {
		return pkgErrors.Wrap(pkgErrors.CodeBadRequest, fmt.Sprintf("无效的仓库名称格式: %s", req.Repo), nil)
	}

	// 2. 查询仓库
	repo, err := s.repoRepo.FindByNamespaceAndName(project, name)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			return pkgErrors.Wrap(pkgErrors.CodeNotFound, fmt.Sprintf("代码库不存在: %s", req.Repo), nil)
		}
		return err
	}

	// 3. 计算构建耗时
	duration := int(req.BuildFinished - req.BuildStarted)

	// 3.1 转换时间戳为 time.Time
	buildCreatedTime := time.Unix(req.BuildCreated, 0)
	buildStartedTime := time.Unix(req.BuildStarted, 0)
	buildFinishedTime := time.Unix(req.BuildFinished, 0)

	// 4. 提取提交者信息（优先级：CommitAuthorName > CommitAuthor > GitAuthorName）
	commitAuthor := req.CommitAuthorName
	if commitAuthor == "" {
		commitAuthor = req.CommitAuthor
	}
	if commitAuthor == "" {
		commitAuthor = req.GitAuthorName
	}

	// 6. 逐个处理应用
	successCount := 0
	var failedApps []string

	for _, appReq := range req.Apps {
		if err := s.processAppBuild(repo, appReq, func(build *model.Build) {
			build.BuildStatus = req.BuildStatus
			build.BuildEvent = req.BuildEvent
			build.BuildLink = req.BuildLink
			build.CommitSHA = req.CommitID
			build.CommitRef = req.CommitRef
			build.CommitBranch = req.CommitBranch
			build.CommitMessage = req.CommitMessage
			build.CommitLink = req.CommitLink

			build.CommitAuthor = commitAuthor
			build.BuildCreated = buildCreatedTime
			build.BuildStarted = buildStartedTime
			build.BuildFinished = buildFinishedTime
			build.BuildDuration = duration
		}); err != nil {
			logger.Error("处理应用构建失败", zap.String("app", appReq.Name), zap.Error(err))
			failedApps = append(failedApps, appReq.Name)
		} else {
			successCount++
		}
	}

	log.With(zap.Int64("build_number", req.BuildNumber)).Infof("构建通知处理完成, success: %d, failed: %v", successCount, utils.Condexpr(len(failedApps) > 0, failedApps, len(failedApps)))

	if len(failedApps) > 0 {
		return pkgErrors.Wrap(pkgErrors.CodePartialSuccess, fmt.Sprintf("部分应用处理失败: %s", strings.Join(failedApps, ", ")), nil)
	}

	return nil
}

// processAppBuild 处理单个应用的构建记录
func (s *buildService) processAppBuild(repo *model.Repository, appReq dto.BuildNotifyApp, updateFunc func(build *model.Build)) error {
	// 1. 查询应用（按 repo_id + name 查询，确保唯一性）
	app, err := s.appRepo.FindByRepoIDAndName(repo.ID, appReq.Name)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			return pkgErrors.Wrap(pkgErrors.CodeNotFound, fmt.Sprintf("应用不存在: %s/%s", repo.Name, appReq.Name), nil)
		}
		return err
	}

	// 2. 检查应用是否属于该仓库
	if app.RepoID != repo.ID {
		return pkgErrors.Wrap(pkgErrors.CodeBadRequest, fmt.Sprintf("应用 %s 不属于仓库 %s", appReq.Name, repo.Name), nil)
	}

	// 3. 构建镜像地址（如果未提供）
	imageURL := ""
	if appReq.Image != nil && *appReq.Image != "" {
		imageURL = *appReq.Image
	}
	// 否则可以从配置中构建，暂时留空

	// 4. 检查构建是否成功（默认为 true）
	buildSuccess := true
	if appReq.BuildSuccess != nil {
		buildSuccess = *appReq.BuildSuccess
	}

	// 5. 创建构建记录
	build := &model.Build{
		RepoID:          repo.ID,
		AppID:           app.ID,
		ImageTag:        appReq.ImageTag,
		ImageURL:        imageURL,
		AppBuildSuccess: buildSuccess,
	}
	updateFunc(build)

	if err := s.buildRepo.Create(build); err != nil {
		return err
	}

	// 6. 通知New Tag事件
	s.coreEngine.NewTag(app.ID, build)

	logger.Info("应用构建记录已创建", zap.Int64("build_id", build.ID), zap.Int64("app_id", app.ID), zap.String("app_name", app.Name), zap.String("tag", appReq.ImageTag))

	return nil
}

// GetByID 根据ID获取构建记录
func (s *buildService) GetByID(id int64) (*dto.BuildResponse, error) {
	build, err := s.buildRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(build), nil
}

// GetByAppAndNumber 根据应用ID和构建号获取构建记录
func (s *buildService) GetByAppAndNumber(appID int64, buildNumber int) (*dto.BuildResponse, error) {
	build, err := s.buildRepo.FindByAppAndNumber(appID, buildNumber)
	if err != nil {
		return nil, err
	}
	return s.toResponse(build), nil
}

// List 查询构建记录列表
func (s *buildService) List(query *dto.BuildListQuery) ([]*dto.BuildResponse, int64, error) {
	builds, total, err := s.buildRepo.List(
		query.GetPage(),
		query.GetPageSize(),
		query.RepoID,
		query.AppID,
		query.BuildStatus,
		query.BuildEvent,
		query.ImageTag,
		query.CommitSHA,
		query.Environment,
		query.Keyword,
	)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.BuildResponse, 0, len(builds))
	for _, build := range builds {
		responses = append(responses, s.toResponse(build))
	}

	return responses, total, nil
}

// ListByRepoID 查询某个仓库的最近构建记录
func (s *buildService) ListByRepoID(repoID int64, limit int) ([]*dto.BuildResponse, error) {
	builds, err := s.buildRepo.ListByRepoID(repoID, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.BuildResponse, 0, len(builds))
	for _, build := range builds {
		responses = append(responses, s.toResponse(build))
	}

	return responses, nil
}

// ListByAppID 查询某个应用的最近构建记录
func (s *buildService) ListByAppID(appID int64, limit int) ([]*dto.BuildResponse, error) {
	builds, err := s.buildRepo.ListByAppID(appID, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.BuildResponse, 0, len(builds))
	for _, build := range builds {
		responses = append(responses, s.toResponse(build))
	}

	return responses, nil
}

// toResponse 转换为响应对象
func (s *buildService) toResponse(build *model.Build) *dto.BuildResponse {
	resp := &dto.BuildResponse{
		ID:              build.ID,
		RepoID:          build.RepoID,
		AppID:           build.AppID,
		BuildNumber:     build.BuildNumber,
		BuildStatus:     build.BuildStatus,
		BuildEvent:      build.BuildEvent,
		BuildLink:       build.BuildLink,
		CommitSHA:       build.CommitSHA,
		CommitRef:       build.CommitRef,
		CommitBranch:    build.CommitBranch,
		CommitMessage:   build.CommitMessage,
		CommitLink:      build.CommitLink,
		CommitAuthor:    build.CommitAuthor,
		BuildCreated:    build.BuildCreated.Format(time.RFC3339),
		BuildStarted:    build.BuildStarted.Format(time.RFC3339),
		BuildFinished:   build.BuildFinished.Format(time.RFC3339),
		BuildDuration:   build.BuildDuration,
		ImageTag:        build.ImageTag,
		ImageURL:        build.ImageURL,
		AppBuildSuccess: build.AppBuildSuccess,
		Environment:     build.Environment,
		CreatedAt:       build.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       build.UpdatedAt.Format(time.RFC3339),
	}

	// 关联的仓库名称
	if build.Repository != nil {
		fullName := fmt.Sprintf("%s/%s", build.Repository.Namespace, build.Repository.Name)
		resp.RepoName = &fullName
	}

	// 关联的应用名称
	if build.Application != nil {
		resp.AppName = &build.Application.Name
	}

	return resp
}

// parseRepo 解析仓库名称
// zkme/zkme-kyb -> (zkme, zkme-kyb)
func parseRepo(repo string) (project, name string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", repo
}
