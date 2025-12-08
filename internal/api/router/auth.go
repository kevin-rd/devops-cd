package router

import (
	"devops-cd/internal/pkg/auth"
	"github.com/gin-gonic/gin"

	"devops-cd/internal/service"
)

var authz service.AuthorizationService

func ProjectAuthWrapper(handler func(c *gin.Context, canAccess func(username string, projectId int64) bool), permission auth.Permission) func(*gin.Context) {
	return func(context *gin.Context) {
		username := context.GetString("username")
		authProvider := context.GetString("auth_type")

		handler(context, func(_ string, projectID int64) bool {
			return authz.CanAccessProject(username, authProvider, projectID, permission)
		})
	}
}

func TeamAuthWrapper(handler func(c *gin.Context, canAccess func(username string, teamID int64) bool), permission auth.Permission) func(*gin.Context) {
	return func(context *gin.Context) {
		username := context.GetString("username")
		authProvider := context.GetString("auth_type")

		handler(context, func(_ string, teamID int64) bool {
			ok, _ := authz.HasTeamPermission(username, authProvider, teamID, permission)
			return ok
		})
	}
}
