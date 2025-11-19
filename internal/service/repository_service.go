package service

import (
	"fmt"
	"time"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	"devops-cd/pkg/constants"
	pkgErrors "devops-cd/pkg/errors"
)

type RepositoryService interface {
	Create(req *dto.CreateRepositoryRequest) (*dto.RepositoryResponse, error)
	GetByID(id int64) (*dto.RepositoryResponse, error)
	List(query *dto.RepositoryListQuery) ([]*dto.RepositoryResponse, int64, error)
	Update(id int64, req *dto.UpdateRepositoryRequest) (*dto.RepositoryResponse, error)
	Delete(id int64) error
}

type repositoryService struct {
	repo    repository.RepositoryRepository
	appRepo repository.ApplicationRepository
}

func NewRepositoryService(repo repository.RepositoryRepository, appRepo repository.ApplicationRepository) RepositoryService {
	return &repositoryService{
		repo:    repo,
		appRepo: appRepo,
	}
}

func (s *repositoryService) Create(req *dto.CreateRepositoryRequest) (*dto.RepositoryResponse, error) {
	// 检查命名空间+仓库名是否已存在
	existing, _ := s.repo.FindByNamespaceAndName(req.Namespace, req.Name)
	if existing != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("代码库 %s/%s 已存在", req.Namespace, req.Name), nil)
	}

	// 创建模型
	repo := &model.Repository{
		Namespace:   req.Namespace,
		Name:        req.Name,
		Description: req.Description,
		GitURL:      req.GitURL,
		GitType:     req.GitType,
		Language:    req.Language,
		TeamID:      req.TeamID,
		ProjectID:   req.ProjectID,
		BaseStatus: model.BaseStatus{
			Status: constants.StatusEnabled,
		},
	}

	// 保存到数据库
	if err := s.repo.Create(repo); err != nil {
		return nil, err
	}

	return s.toResponse(repo, nil), nil
}

func (s *repositoryService) GetByID(id int64) (*dto.RepositoryResponse, error) {
	repo, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 加载该代码库下的应用列表
	apps, err := s.appRepo.ListByRepoID(id)
	if err != nil {
		// 如果加载应用列表失败，仍然返回代码库信息，但应用列表为空
		// 这样即使application表有问题也不影响repository查询
		return s.toResponse(repo, nil), nil
	}

	return s.toResponse(repo, apps), nil
}

func (s *repositoryService) List(query *dto.RepositoryListQuery) ([]*dto.RepositoryResponse, int64, error) {
	repos, total, err := s.repo.List(
		query.GetPage(),
		query.GetPageSize(),
		query.Namespace,
		query.TeamID,
		query.GitType,
		query.Keyword,
		query.Status,
	)
	if err != nil {
		return nil, 0, err
	}

	// 判断是否需要加载应用列表
	withApps := query.WithApplications != nil && *query.WithApplications

	responses := make([]*dto.RepositoryResponse, len(repos))
	for i, repo := range repos {
		if withApps {
			// 加载该代码库的应用列表
			apps, err := s.appRepo.ListByRepoID(repo.ID)
			if err != nil {
				// 如果加载失败，返回空应用列表
				responses[i] = s.toResponse(repo, nil)
			} else {
				responses[i] = s.toResponse(repo, apps)
			}
		} else {
			// 默认不加载应用列表，避免N+1查询问题
			responses[i] = s.toResponse(repo, nil)
		}
	}

	return responses, total, nil
}

func (s *repositoryService) Update(id int64, req *dto.UpdateRepositoryRequest) (*dto.RepositoryResponse, error) {
	// 查询代码库
	repo, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 检查命名空间+仓库名是否冲突
	if (req.Namespace != nil && *req.Namespace != repo.Namespace) || (req.Name != nil && *req.Name != repo.Name) {
		checkNamespace := repo.Namespace
		checkName := repo.Name
		if req.Namespace != nil {
			checkNamespace = *req.Namespace
		}
		if req.Name != nil {
			checkName = *req.Name
		}

		existing, _ := s.repo.FindByNamespaceAndName(checkNamespace, checkName)
		if existing != nil && existing.ID != id {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("代码库 %s/%s 已存在", checkNamespace, checkName), nil)
		}
	}

	// 更新字段
	if req.Namespace != nil {
		repo.Namespace = *req.Namespace
	}
	if req.Name != nil {
		repo.Name = *req.Name
	}
	if req.Description != nil {
		repo.Description = req.Description
	}
	if req.GitURL != nil {
		repo.GitURL = *req.GitURL
	}
	if req.GitType != nil {
		repo.GitType = *req.GitType
	}
	if req.Language != nil {
		repo.Language = req.Language
	}
	if req.TeamID != nil {
		repo.TeamID = req.TeamID
	}
	if req.ProjectID != nil {
		repo.ProjectID = req.ProjectID
	}
	if req.Status != nil {
		repo.Status = *req.Status
	}

	// 保存更新
	if err := s.repo.Update(repo); err != nil {
		return nil, err
	}

	return s.toResponse(repo, nil), nil
}

func (s *repositoryService) Delete(id int64) error {
	// 检查代码库是否存在
	_, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// 删除代码库
	return s.repo.Delete(id)
}

// toResponse 转换为响应对象
func (s *repositoryService) toResponse(repo *model.Repository, apps []*model.Application) *dto.RepositoryResponse {
	resp := &dto.RepositoryResponse{
		ID:          repo.ID,
		Namespace:   repo.Namespace,
		Name:        repo.Name,
		FullName:    fmt.Sprintf("%s/%s", repo.Namespace, repo.Name),
		Description: repo.Description,
		GitURL:      repo.GitURL,
		GitType:     repo.GitType,
		Language:    repo.Language,
		TeamID:      repo.TeamID,
		ProjectID:   repo.ProjectID,
		Status:      repo.Status,
		CreatedAt:   repo.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   repo.UpdatedAt.Format(time.RFC3339),
	}

	// 添加团队名称
	if repo.Team != nil {
		resp.TeamName = &repo.Team.Name
	}

	// 添加项目名称（优先使用 DisplayName，否则使用 Name）
	if repo.Project != nil {
		if repo.Project.DisplayName != nil && *repo.Project.DisplayName != "" {
			resp.ProjectName = repo.Project.DisplayName
		} else {
			resp.ProjectName = &repo.Project.Name
		}
	}

	// 添加应用列表
	if len(apps) > 0 {
		resp.Applications = make([]*dto.ApplicationResponse, len(apps))
		for i, app := range apps {
			resp.Applications[i] = s.toApplicationResponse(app)
		}
	}

	return resp
}

// toApplicationResponse 转换应用为响应对象（简化版）
func (s *repositoryService) toApplicationResponse(app *model.Application) *dto.ApplicationResponse {
	appResp := &dto.ApplicationResponse{
		ID:          app.ID,
		Name:        app.Name,
		Namespace:   app.Namespace,
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
		repoName := fmt.Sprintf("%s/%s", app.Repository.Namespace, app.Repository.Name)
		appResp.RepoName = &repoName
	}

	// 添加团队名称
	if app.Team != nil {
		appResp.TeamName = &app.Team.Name
	}

	return appResp
}
