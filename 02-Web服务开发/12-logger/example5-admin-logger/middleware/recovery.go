package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecoveryMiddleware 错误恢复中间件
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := c.GetString("request_id")
				stack := string(debug.Stack())

				// 记录 panic
				logger.Error("panic recovered",
					zap.String("request_id", requestID),
					zap.Any("error", err),
					zap.String("stack", stack),
				)

				// 返回错误响应
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    10000,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
