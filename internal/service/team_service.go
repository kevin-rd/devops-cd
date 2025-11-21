package service

import (
	"fmt"
	"time"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	pkgErrors "devops-cd/pkg/errors"
)

type TeamService interface {
	Create(req *dto.CreateTeamRequest) (*dto.TeamResponse, error)
	GetByID(id int64) (*dto.TeamResponse, error)
	List(projectID *int64) ([]*dto.TeamSimpleResponse, error)
	Update(id int64, req *dto.UpdateTeamRequest) (*dto.TeamResponse, error)
	Delete(id int64) error
}

type teamService struct {
	repo        repository.TeamRepository
	projectRepo repository.ProjectRepository
}

func NewTeamService(repo repository.TeamRepository, projectRepo repository.ProjectRepository) TeamService {
	return &teamService{
		repo:        repo,
		projectRepo: projectRepo,
	}
}

func (s *teamService) Create(req *dto.CreateTeamRequest) (*dto.TeamResponse, error) {
	// 检查团队名称是否已存在
	existing, _ := s.repo.FindByName(req.Name)
	if existing != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("团队 %s 已存在", req.Name), nil)
	}

	if _, err := s.projectRepo.FindByID(req.ProjectID); err != nil {
		return nil, err
	}

	team := &model.Team{
		Name:        req.Name,
		ProjectID:   req.ProjectID,
		Description: req.Description,
		LeaderName:  req.LeaderName,
	}

	if err := s.repo.Create(team); err != nil {
		return nil, err
	}

	return s.toResponse(team), nil
}

func (s *teamService) GetByID(id int64) (*dto.TeamResponse, error) {
	team, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(team), nil
}

func (s *teamService) List(projectID *int64) ([]*dto.TeamSimpleResponse, error) {
	var teams []*model.Team
	var err error

	if projectID != nil {
		teams, err = s.repo.ListByProjectID(*projectID)
	} else {
		teams, err = s.repo.ListAll()
	}

	if err != nil {
		return nil, err
	}

	responses := make([]*dto.TeamSimpleResponse, len(teams))
	for i, team := range teams {
		responses[i] = &dto.TeamSimpleResponse{
			ID:        team.ID,
			Name:      team.Name,
			ProjectID: team.ProjectID,
		}
	}

	return responses, nil
}

func (s *teamService) Update(id int64, req *dto.UpdateTeamRequest) (*dto.TeamResponse, error) {
	// 查询团队
	team, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 检查名称是否冲突
	if req.Name != nil && *req.Name != team.Name {
		existing, _ := s.repo.FindByName(*req.Name)
		if existing != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("团队 %s 已存在", *req.Name), nil)
		}
		team.Name = *req.Name
	}

	// 更新字段
	if req.ProjectID != nil && *req.ProjectID != team.ProjectID {
		if _, err := s.projectRepo.FindByID(*req.ProjectID); err != nil {
			return nil, err
		}
		team.ProjectID = *req.ProjectID
	}
	if req.Description != nil {
		team.Description = req.Description
	}
	if req.LeaderName != nil {
		team.LeaderName = req.LeaderName
	}

	// 保存更新
	if err := s.repo.Update(team); err != nil {
		return nil, err
	}

	return s.toResponse(team), nil
}

func (s *teamService) Delete(id int64) error {
	// 检查团队是否存在
	_, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// TODO: 检查是否有代码库或应用关联此团队，如果有则提示用户先解除关联

	// 软删除团队
	return s.repo.Delete(id)
}

// toResponse 转换为响应对象
func (s *teamService) toResponse(team *model.Team) *dto.TeamResponse {
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
