package service

import (
	"fmt"
	"time"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	pkgErrors "devops-cd/pkg/errors"
)

type ProjectService interface {
	Create(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	GetByID(id int64) (*dto.ProjectResponse, error)
	List(query *dto.ProjectListQuery) ([]*dto.ProjectResponse, int64, error)
	ListAll() ([]*dto.ProjectSimpleResponse, error)
	Update(id int64, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	Delete(id int64) error
}

type projectService struct {
	repo repository.ProjectRepository
}

func NewProjectService(repo repository.ProjectRepository) ProjectService {
	return &projectService{
		repo: repo,
	}
}

func (s *projectService) Create(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	// 检查项目名称是否已存在
	existing, _ := s.repo.FindByName(req.Name)
	if existing != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("项目 %s 已存在", req.Name), nil)
	}

	// 创建项目
	project := &model.Project{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		OwnerName:   req.OwnerName,
	}

	if err := s.repo.Create(project); err != nil {
		return nil, err
	}

	return s.toResponse(project), nil
}

func (s *projectService) GetByID(id int64) (*dto.ProjectResponse, error) {
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(project), nil
}

func (s *projectService) List(query *dto.ProjectListQuery) ([]*dto.ProjectResponse, int64, error) {
	projects, total, err := s.repo.List(
		query.GetPage(),
		query.GetPageSize(),
		query.Keyword,
	)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.ProjectResponse, len(projects))
	for i, project := range projects {
		responses[i] = s.toResponse(project)
	}

	return responses, total, nil
}

func (s *projectService) ListAll() ([]*dto.ProjectSimpleResponse, error) {
	projects, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.ProjectSimpleResponse, len(projects))
	for i, project := range projects {
		responses[i] = &dto.ProjectSimpleResponse{
			ID:          project.ID,
			Name:        project.Name,
			DisplayName: project.DisplayName,
		}
	}

	return responses, nil
}

func (s *projectService) Update(id int64, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	// 查询项目
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 检查名称是否冲突
	if req.Name != nil && *req.Name != project.Name {
		existing, _ := s.repo.FindByName(*req.Name)
		if existing != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("项目 %s 已存在", *req.Name), nil)
		}
		project.Name = *req.Name
	}

	// 更新字段
	if req.DisplayName != nil {
		project.DisplayName = req.DisplayName
	}
	if req.Description != nil {
		project.Description = req.Description
	}
	if req.OwnerName != nil {
		project.OwnerName = req.OwnerName
	}

	// 保存更新
	if err := s.repo.Update(project); err != nil {
		return nil, err
	}

	return s.toResponse(project), nil
}

func (s *projectService) Delete(id int64) error {
	// 检查项目是否存在
	_, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// TODO: 检查是否有代码库关联此项目，如果有则提示用户先解除关联

	// 软删除项目
	return s.repo.Delete(id)
}

// toResponse 转换为响应对象
func (s *projectService) toResponse(project *model.Project) *dto.ProjectResponse {
	return &dto.ProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		DisplayName: project.DisplayName,
		Description: project.Description,
		OwnerName:   project.OwnerName,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	}
}
