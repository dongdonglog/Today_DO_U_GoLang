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

// UserRole 用户角色
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
	RoleGuest UserRole = "guest"
)

// User 用户模型
// @Description 用户信息
type User struct {
	ID        int      `json:"id" example:"1"`
	Name      string   `json:"name" example:"Alice" binding:"required,min=2,max=50"`
	Email     string   `json:"email" example:"alice@example.com" binding:"required,email"`
	Role      UserRole `json:"role" example:"user"`
	CreatedAt string   `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// Pagination 分页信息
type Pagination struct {
	Page  int `json:"page" example:"1"`
	Size  int `json:"size" example:"10"`
	Total int `json:"total" example:"100"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Code       int        `json:"code" example:"0"`
	Message    string     `json:"message" example:"success"`
	Data       []User     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code" example:"10001"`
	Message string `json:"message" example:"invalid request"`
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
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/users [get]
func listUsersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, UserListResponse{
		Code:    0,
		Message: "success",
		Data: []User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Role: RoleAdmin, CreatedAt: "2024-01-15T10:30:00Z"},
			{ID: 2, Name: "Bob", Email: "bob@example.com", Role: RoleUser, CreatedAt: "2024-01-16T11:30:00Z"},
		},
		Pagination: Pagination{
			Page:  1,
			Size:  10,
			Total: 2,
		},
	})
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param user body User true "用户信息"
// @Success 201 {object} User
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/users [post]
func createUserHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    10001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func main() {
	r := gin.Default()

	r.GET("/api/v1/users", listUsersHandler)
	r.POST("/api/v1/users", createUserHandler)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8080")
}
