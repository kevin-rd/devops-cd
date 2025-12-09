package middleware

import (
	"devops-cd/pkg/responses"
	"strings"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/jwt"
	"devops-cd/pkg/constants"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader(constants.HeaderAuthorization)
		if authHeader == "" {
			responses.ErrorWithCode(c, 401, "缺少Authorization Header")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, constants.HeaderBearerPrefix) {
			responses.ErrorWithCode(c, 401, "Authorization格式错误")
			c.Abort()
			return
		}

		// 提取Token
		token := strings.TrimPrefix(authHeader, constants.HeaderBearerPrefix)

		// 验证Token
		claims, err := jwt.ValidateToken(token)
		if err != nil {
			responses.Error(c, err)
			c.Abort()
			return
		}

		// 检查Token类型(必须是AccessToken)
		if claims.Type != constants.JWTTypeAccess {
			responses.ErrorWithCode(c, 401, "无效的Token类型")
			c.Abort()
			return
		}

		// 将用户信息存入context
		userInfo := &dto.UserInfo{
			Username:    claims.Username,
			Email:       claims.Email,
			DisplayName: claims.DisplayName,
			AuthType:    claims.AuthType,
			UID:         claims.UID,
			Phone:       claims.Phone,
		}
		c.Set("user", userInfo)
		c.Set("username", claims.Username)
		c.Set("auth_type", claims.AuthType)
		c.Set("uid", claims.UID)
		c.Set("phone", claims.Phone)

		c.Next()
	}
}
