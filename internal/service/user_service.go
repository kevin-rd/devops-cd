package service

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/auth"
	"devops-cd/internal/repository"
)

type UserService interface {
	Search(req *dto.UserSearchQuery) ([]*dto.UserSimpleResponse, int64, error)
	ListRoles() []string
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) Search(req *dto.UserSearchQuery) ([]*dto.UserSimpleResponse, int64, error) {
	page := req.GetPage()
	pageSize := req.GetPageSize()
	// 默认 20，最大 20（符合前端预期 10/20）
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 20 {
		pageSize = 20
	}

	users, total, err := s.userRepo.Search(req.Keyword, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*dto.UserSimpleResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, &dto.UserSimpleResponse{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Email:       u.Email,
		})
	}

	return resp, total, nil
}

func (s *userService) ListRoles() []string {
	// 按固定顺序返回
	return []string{
		string(auth.RoleSystemAdmin),
		string(auth.RoleSystemViewer),
		string(auth.RoleProjectAdmin),
		string(auth.RoleProjectViewer),
		string(auth.RoleTeamAdmin),
		string(auth.RoleMember),
	}
}
