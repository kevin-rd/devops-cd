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
	repo     repository.ProjectRepository
	teamRepo repository.TeamRepository
}

func NewProjectService(repo repository.ProjectRepository, teamRepo repository.TeamRepository) ProjectService {
	return &projectService{
		repo:     repo,
		teamRepo: teamRepo,
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
		Description: req.Description,
		OwnerName:   req.OwnerName,
	}

	if err := s.repo.Create(project); err != nil {
		return nil, err
	}

	if s.shouldCreateDefaultTeam(req) {
		team := &model.Team{
			Name:      project.Name,
			ProjectID: project.ID,
		}
		if err := s.teamRepo.Create(team); err != nil {
			// 回滚项目创建
			_ = s.repo.Delete(project.ID)
			return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "创建默认团队失败", err)
		}
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

	var teamMap map[int64][]*dto.TeamResponse
	if query.WithTeams && len(projects) > 0 {
		projectIDs := make([]int64, len(projects))
		for i, project := range projects {
			projectIDs[i] = project.ID
		}
		teams, err := s.teamRepo.ListByProjectIDs(projectIDs)
		if err != nil {
			return nil, 0, err
		}
		teamMap = make(map[int64][]*dto.TeamResponse)
		for _, team := range teams {
			teamMap[team.ProjectID] = append(teamMap[team.ProjectID], s.toTeamResponse(team))
		}
	}

	for i, project := range projects {
		resp := s.toResponse(project)
		if teamMap != nil {
			resp.Teams = teamMap[project.ID]
		}
		responses[i] = resp
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
			ID:   project.ID,
			Name: project.Name,
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
		Description: project.Description,
		OwnerName:   project.OwnerName,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *projectService) toTeamResponse(team *model.Team) *dto.TeamResponse {
	return &dto.TeamResponse{
		ID:          team.ID,
		Name:        team.Name,
		ProjectID:   team.ProjectID,
		Description: team.Description,
		LeaderName:  team.LeaderName,
		CreatedAt:   team.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   team.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *projectService) shouldCreateDefaultTeam(req *dto.CreateProjectRequest) bool {
	if req.CreateDefaultTeam == nil {
		return true
	}
	return *req.CreateDefaultTeam
}
