package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		log.Printf("[%d] %s %s %v", statusCode, c.Request.Method, path, latency)
	}
}

// Auth 认证中间件
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization token",
			})
			c.Abort()
			return
		}

		if token != "Bearer secret-token" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid token",
			})
			c.Abort()
			return
		}

		// 认证成功，继续处理
		c.Next()
	}
}

func main() {
	r := gin.New()

	// 全局中间件
	r.Use(Logger())
	r.Use(gin.Recovery())

	// 公开接口
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "public endpoint",
		})
	})

	// 需要认证的接口
	auth := r.Group("/api")
	auth.Use(Auth())
	{
		auth.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "protected endpoint",
			})
		})

		auth.GET("/user", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"user": "Alice",
			})
		})
	}

	r.Run(":8080")
}
