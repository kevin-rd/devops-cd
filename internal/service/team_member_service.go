package service

import (
	"strings"
	"time"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	pkgErrors "devops-cd/pkg/errors"
)

const defaultTeamMemberRole = "member"

type TeamMemberService interface {
	Add(req *dto.TeamMemberAddRequest) (*dto.TeamMemberResponse, error)
	List(req *dto.TeamMemberListQuery) ([]*dto.TeamMemberResponse, int64, error)
	UpdateRole(id int64, req *dto.TeamMemberUpdateRoleRequest) (*dto.TeamMemberResponse, error)
	Remove(id int64) error
}

type teamMemberService struct {
	repo     repository.TeamMemberRepository
	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
}

func NewTeamMemberService(repo repository.TeamMemberRepository, teamRepo repository.TeamRepository, userRepo repository.UserRepository) TeamMemberService {
	return &teamMemberService{
		repo:     repo,
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

func (s *teamMemberService) Add(req *dto.TeamMemberAddRequest) (*dto.TeamMemberResponse, error) {
	if _, err := s.teamRepo.FindByID(req.TeamID); err != nil {
		return nil, err
	}
	user, err := s.userRepo.FindByID(req.UserID)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.FindByTeamAndUser(req.TeamID, req.UserID); err == nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeConflict, "成员已存在", nil)
	} else if err != pkgErrors.ErrRecordNotFound {
		return nil, err
	}

	role := defaultTeamMemberRole
	if req.Role != nil && strings.TrimSpace(*req.Role) != "" {
		role = strings.TrimSpace(*req.Role)
	}

	member := &model.TeamMember{
		TeamID: req.TeamID,
		UserID: req.UserID,
		Role:   role,
	}

	if err := s.repo.Create(member); err != nil {
		return nil, err
	}
	member.User = user

	return s.toResponse(member), nil
}

func (s *teamMemberService) List(req *dto.TeamMemberListQuery) ([]*dto.TeamMemberResponse, int64, error) {
	members, total, err := s.repo.ListByTeam(req.TeamID, req.GetPage(), req.GetPageSize(), req.Keyword)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.TeamMemberResponse, len(members))
	for i, member := range members {
		responses[i] = s.toResponse(member)
	}
	return responses, total, nil
}

func (s *teamMemberService) UpdateRole(id int64, req *dto.TeamMemberUpdateRoleRequest) (*dto.TeamMemberResponse, error) {
	member, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	member.Role = strings.TrimSpace(req.Role)
	if member.Role == "" {
		member.Role = defaultTeamMemberRole
	}

	if err := s.repo.Update(member); err != nil {
		return nil, err
	}

	return s.toResponse(member), nil
}

func (s *teamMemberService) Remove(id int64) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *teamMemberService) toResponse(member *model.TeamMember) *dto.TeamMemberResponse {
	resp := &dto.TeamMemberResponse{
		ID:        member.ID,
		TeamID:    member.TeamID,
		UserID:    member.UserID,
		Role:      member.Role,
		CreatedAt: member.CreatedAt.Format(time.RFC3339),
		UpdatedAt: member.UpdatedAt.Format(time.RFC3339),
	}
	if member.User != nil {
		resp.Username = member.User.Username
		resp.DisplayName = member.User.DisplayName
		resp.Email = member.User.Email
	}
	return resp
}
