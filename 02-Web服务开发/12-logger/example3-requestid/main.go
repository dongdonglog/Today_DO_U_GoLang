package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// contextKey 上下文键类型
type contextKey string

const (
	// RequestIDKey 请求 ID 键
	RequestIDKey contextKey = "request_id"
)

// WithRequestID 将 requestID 存入 context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// RequestIDFromContext 从 context 获取 requestID
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}

// RequestIDMiddleware 请求 ID 中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 requestID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成新的 requestID
			requestID = uuid.New().String()
		}

		// 存入 gin context
		c.Set("request_id", requestID)

		// 存入 Go context
		ctx := WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// 设置响应头
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		requestID := c.GetString("request_id")

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()

		logger.Info("request completed",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		)
	}
}

func main() {
	// 创建 logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 创建路由
	r := gin.New()

	// 使用中间件
	r.Use(RequestIDMiddleware())
	r.Use(LoggerMiddleware(logger))

	// 路由
	r.GET("/health", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		logger.Info("health check",
			zap.String("request_id", requestID),
		)
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"request_id": requestID,
		})
	})

	r.POST("/users", func(c *gin.Context) {
		requestID := c.GetString("request_id")

		// 模拟业务逻辑
		userID := 123
		logger.Info("user created",
			zap.String("request_id", requestID),
			zap.Int("user_id", userID),
		)

		c.JSON(http.StatusCreated, gin.H{
			"request_id": requestID,
			"user_id":    userID,
			"name":       "Alice",
		})
	})

	// 启动服务器
	fmt.Println("Server starting on :8080")
	fmt.Println("Test commands:")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("  curl -X POST http://localhost:8080/users")
	fmt.Println("  curl -H 'X-Request-ID: custom-123' http://localhost:8080/health")

	r.Run(":8080")
}
