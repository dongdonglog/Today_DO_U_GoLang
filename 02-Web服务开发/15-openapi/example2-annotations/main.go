package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Admin API
// @version 1.0
// @description 后台管理系统 API 文档
// @host localhost:8080
// @BasePath /

// User 用户模型
type User struct {
	ID    int    `json:"id" example:"1"`
	Name  string `json:"name" example:"Alice"`
	Email string `json:"email" example:"alice@example.com"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Code    int    `json:"code" example:"0"`
	Message string `json:"message" example:"success"`
	Data    []User `json:"data"`
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表，支持分页
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} UserListResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/users [get]
func listUsersHandler(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	size := c.DefaultQuery("size", "10")

	c.JSON(http.StatusOK, UserListResponse{
		Code:    0,
		Message: "success",
		Data: []User{
			{ID: 1, Name: "Alice", Email: "alice@example.com"},
			{ID: 2, Name: "Bob", Email: "bob@example.com"},
		},
	})

	_ = page
	_ = size
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param user body User true "用户信息"
// @Success 201 {object} User
// @Failure 400 {object} map[string]string
// @Router /api/v1/users [post]
func createUserHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func main() {
	r := gin.Default()

	r.GET("/api/v1/users", listUsersHandler)
	r.POST("/api/v1/users", createUserHandler)

	// 注册 Swagger 路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8080")
}
