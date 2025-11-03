package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devops-cd/internal/pkg/logger"
)

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		cost := time.Since(start)

		logger.Info(fmt.Sprintf("%s %s %s %v %.2fs %v", c.Request.Proto, c.Request.Method, path, c.Writer.Status(), cost.Seconds(), query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		)
	}
}
