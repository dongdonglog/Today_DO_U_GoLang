package main

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestIDMiddleware 请求 ID 中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggerMiddleware 请求日志中间件
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 记录请求开始
		logger.Info("request started",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		)

		c.Next()

		// 记录请求完成
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()

		// 根据状态码选择日志级别
		var logFunc func(string, ...zap.Field)
		if statusCode >= 500 {
			logFunc = logger.Error
		} else if statusCode >= 400 {
			logFunc = logger.Warn
		} else {
			logFunc = logger.Info
		}

		logFunc("request completed",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.Int("body_size", bodySize),
		)
	}
}

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

// MetricsMiddleware 指标中间件（示例）
func MetricsMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// 这里可以发送指标到 Prometheus 等
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		path := c.Request.URL.Path

		// 示例：记录慢请求
		if latency > 1*time.Second {
			logger.Warn("slow request",
				zap.String("path", path),
				zap.Duration("latency", latency),
				zap.Int("status", statusCode),
			)
		}
	}
}

func main() {
	// 创建 logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 创建路由
	r := gin.New()

	// 中间件顺序很重要
	r.Use(RecoveryMiddleware(logger))  // 1. 错误恢复
	r.Use(RequestIDMiddleware())       // 2. 请求 ID
	r.Use(LoggerMiddleware(logger))    // 3. 请求日志
	r.Use(MetricsMiddleware(logger))   // 4. 指标收集

	// 正常路由
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

	// 模拟 404
	r.GET("/not-found", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "not found",
		})
	})

	// 模拟 500
	r.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "internal error",
		})
	})

	// 模拟 panic
	r.GET("/panic", func(c *gin.Context) {
		panic("something went wrong")
	})

	// 模拟慢请求
	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(2 * time.Second)
		c.JSON(http.StatusOK, gin.H{
			"message": "slow response",
		})
	})

	// 启动服务器
	fmt.Println("Server starting on :8080")
	fmt.Println("Test commands:")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("  curl http://localhost:8080/not-found")
	fmt.Println("  curl http://localhost:8080/error")
	fmt.Println("  curl http://localhost:8080/panic")
	fmt.Println("  curl http://localhost:8080/slow")

	r.Run(":8080")
}
