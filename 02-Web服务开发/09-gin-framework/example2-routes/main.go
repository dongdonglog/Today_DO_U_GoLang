package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 路径参数
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"user_id": id,
		})
	})

	// 查询参数
	r.GET("/users", func(c *gin.Context) {
		page := c.DefaultQuery("page", "1")
		size := c.DefaultQuery("size", "10")
		c.JSON(http.StatusOK, gin.H{
			"page": page,
			"size": size,
		})
	})

	// POST 请求
	r.POST("/users", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{
			"message": "user created",
		})
	})

	// PUT 请求
	r.PUT("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("user %s updated", id),
		})
	})

	// DELETE 请求
	r.DELETE("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("user %s deleted", id),
		})
	})

	// 任意方法
	r.Any("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// 404 处理
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "route not found",
		})
	})

	r.Run(":8080")
}
