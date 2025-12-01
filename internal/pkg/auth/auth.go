package auth

import "strings"

// Role 内置角色
type Role string

const (
	RoleSystemAdmin   Role = "system_admin"
	RoleSystemViewer  Role = "system_viewer"
	RoleProjectAdmin  Role = "project_admin"
	RoleProjectViewer Role = "project_viewer"
	RoleTeamAdmin     Role = "team_admin"
	RoleMember        Role = "team_member"
)

// Permission 内置权限
type Permission string

const (
	PermProjectCreate Permission = "project:create"
	PermProjectDelete Permission = "project:delete"
	PermProjectUpdate Permission = "project:update"

	PermBatchCreate  Permission = "batch:create"
	PermBatchUpdate  Permission = "batch:update"
	PermBatchDelete  Permission = "batch:delete"
	PermBatchFlow    Permission = "batch:action"
	PermBatchView    Permission = "batch:view"
	PermBatchApprove Permission = "batch:approve"

	PermReleaseAppCreate Permission = "batch:release_app:create"
	PermReleaseAppUpdate Permission = "batch:release_app:update"
	PermReleaseAppDelete Permission = "batch:release_app:delete"
)

// RolePermissions 每个角色拥有的权限集合
var RolePermissions = map[Role][]Permission{
	RoleSystemAdmin: {
		"*",
	},
	RoleSystemViewer: {
		"*:view",
	},
	RoleProjectAdmin: {
		"project:*",
		"batch:*",
		"team:*",
	},
	RoleProjectViewer: {
		"project:view",
		"batch:view",
		"team:view",
	},
	RoleTeamAdmin: {
		"batch:*",
		"team:*",
	},
	RoleMember: {
		"batch:*",
	},
}

// Allow 判断一组角色是否包含所需权限，支持通配符
func Allow(roles []string, need Permission) bool {
	permissions := collectPermissions(roles)

	return len(permissions) > 0 && allow(permissions, need)
}

func collectPermissions(roles []string) []Permission {
	perms := make([]Permission, 0)
	for _, r := range roles {
		if ps, ok := RolePermissions[Role(r)]; ok {
			perms = append(perms, ps...)
		}
	}
	return perms
}

func allow(have []Permission, need Permission) bool {
	for _, p := range have {
		if p == need {
			return true
		}

		if p == "*" {
			return true
		}

		reqParts := strings.Split(string(need), ":")
		allParts := strings.Split(string(p), ":")

		// 必须段数一致或 allowed 更短（支持前缀通配）
		if len(allParts) > len(reqParts)+1 || len(allParts) < 1 {
			return false
		}

		for i := 0; i < len(allParts); i++ {
			if i >= len(reqParts) {
				return false // required 已经结束，但 allowed 还有更多段
			}

			if allParts[i] == "*" {
				// * 可以匹配剩余所有段（类似 RESTful 的 /**）
				return true
			}

			if allParts[i] != reqParts[i] {
				return false
			}
		}

		// allowed 已经匹配完，required 可能还有剩余 → 允许以 * 结尾
		return len(allParts) > 0 && allParts[len(allParts)-1] == "*" && len(reqParts) >= len(allParts)-1
	}
	return false
}
