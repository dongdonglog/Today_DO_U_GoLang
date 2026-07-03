package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("[%d] %s %s %v", c.Writer.Status(), c.Request.Method, c.Request.URL.Path, time.Since(start))
	}
}

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token != "Bearer secret-token" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func AdminRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetHeader("X-Role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func main() {
	r := gin.New()
	r.Use(gin.Recovery(), Logger())

	// 公开接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 分组
	v1 := r.Group("/api/v1")
	{
		// 用户相关（需要认证）
		users := v1.Group("/users")
		users.Use(Auth())
		{
			users.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": []string{"Alice", "Bob"}})
			})
			users.GET("/:id", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": c.Param("id")}})
			})
			users.POST("", func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "created"})
			})
		}

		// 订单相关（需要认证）
		orders := v1.Group("/orders")
		orders.Use(Auth())
		{
			orders.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": []string{}})
			})
		}
	}

	// API v2 分组
	v2 := r.Group("/api/v2")
	{
		v2.GET("/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": []gin.H{{"id": 1, "name": "Alice"}}})
		})
	}

	// 管理后台（需要认证 + 管理员角色）
	admin := r.Group("/admin")
	admin.Use(Auth(), AdminRole())
	{
		admin.GET("/dashboard", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"users": 100, "orders": 500}})
		})
		admin.GET("/settings", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"site_name": "My App"}})
		})
	}

	r.Run(":8080")
}
