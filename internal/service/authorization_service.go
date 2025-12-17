package service

import (
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/auth"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/repository"
	"devops-cd/pkg/constants"
	pkgErrors "devops-cd/pkg/responses"
	"errors"
	"github.com/samber/lo"
)

// AuthorizationService 负责基于内置角色/权限做简单的团队权限判断
// 当前实现逻辑：
//  1. 先检查用户的系统级角色（users.system_roles）是否已拥有该权限
//  2. 再检查 team_members 表中该用户在指定 team 下的角色是否拥有该权限
//  3. 角色 -> 权限 的关系写死在 internal/model/auth.go 的 RolePermissions 中
//  4. 权限匹配使用 model.allow，支持通配符（如 view:*、resource:*）
type AuthorizationService interface {
	CanAccessProject(username, authProvider string, projectId int64, perm auth.Permission) bool
	// HasTeamPermission 判断某个用户在指定 team 下是否拥有某个权限
	HasTeamPermission(username, authProvider string, teamID int64, perm auth.Permission) (bool, error)
}

type authorizationService struct {
	userRepo       *repository.UserRepository
	teamMemberRepo *repository.TeamMemberRepository
}

// NewAuthorizationService 创建 AuthorizationService
func NewAuthorizationService(userRepo *repository.UserRepository, teamMemberRepo *repository.TeamMemberRepository) AuthorizationService {
	return &authorizationService{
		userRepo:       userRepo,
		teamMemberRepo: teamMemberRepo,
	}
}

func (s *authorizationService) CanAccessProject(username, authProvider string, projectId int64, perm auth.Permission) bool {

	user, err := s.userRepo.FindWithTeams(username, normalizeProvider(authProvider))
	if err != nil {
		if errors.Is(err, pkgErrors.ErrRecordNotFound) {
			// 用户不存在，视为无权限
			return false
		}
		logger.Sugar().Warnf("find user error: %v", err)
		return false
	}

	// 检查系统级角色权限
	if auth.Allow(user.SystemRoles, perm) {
		return true
	}

	roles := lo.Uniq(lo.FlatMap(user.TeamMembers, func(t model.TeamMember, _ int) []string { return t.Roles }))
	return auth.Allow(roles, perm)
}

// HasTeamPermission 权限判断核心逻辑
//
// 1. 按 username 查询用户信息（本地 users 表）
// 2. 基于用户的 SystemRoles 计算系统级权限，如果已满足则直接放行
// 3. 否则查询 team_members 中该用户在指定 team 下的记录，基于成员角色计算权限
// 4. 最终使用 model.allow 进行权限匹配
func (s *authorizationService) HasTeamPermission(username, authProvider string, teamID int64, perm auth.Permission) (bool, error) {
	// 1. 查询用户
	user, err := s.userRepo.FindByUsername(username, normalizeProvider(authProvider))
	if err != nil {
		if errors.Is(err, pkgErrors.ErrRecordNotFound) {
			// 用户不存在，视为无权限
			return false, nil
		}
		logger.Warn("")
		return false, err
	}

	// 2. 系统级角色权限检查（users.system_roles）
	if auth.Allow(user.SystemRoles, perm) {
		return true, nil
	}

	// 3. team 内角色权限检查（team_members.roles）
	member, err := s.teamMemberRepo.FindByTeamAndUser(teamID, user.ID)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			// 未加入该团队，视为无权限
			return false, nil
		}
		return false, err
	}

	return auth.Allow(member.Roles, perm), nil
}

func normalizeProvider(provider string) string {
	if provider == "" {
		return constants.AuthTypeLocal
	}
	return provider
}
