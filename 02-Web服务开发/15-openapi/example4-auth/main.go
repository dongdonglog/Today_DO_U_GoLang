package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Admin API
// @version 1.0
// @description 后台管理系统 API 文档
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer {token}

// User 用户模型
type User struct {
	ID    int    `json:"id" example:"1"`
	Name  string `json:"name" example:"Alice"`
	Email string `json:"email" example:"alice@example.com"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" example:"admin" binding:"required"`
	Password string `json:"password" example:"admin123" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Code    int    `json:"code" example:"0"`
	Message string `json:"message" example:"success"`
	Token   string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code" example:"10002"`
	Message string `json:"message" example:"unauthorized"`
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录，获取 Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body LoginRequest true "登录信息"
// @Success 200 {object} LoginResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/login [post]
func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    10002,
			Message: "invalid credentials",
		})
		return
	}

	// 模拟登录
	if req.Username == "admin" && req.Password == "admin123" {
		c.JSON(http.StatusOK, LoginResponse{
			Code:    0,
			Message: "success",
			Token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		})
		return
	}

	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Code:    10002,
		Message: "invalid credentials",
	})
}

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:    10002,
				Message: "missing token",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:    10002,
				Message: "invalid token format",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser 获取当前用户
// @Summary 获取当前用户
// @Description 获取当前登录用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} User
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/me [get]
func getCurrentUserHandler(c *gin.Context) {
	c.JSON(http.StatusOK, User{
		ID:    1,
		Name:  "Alice",
		Email: "alice@example.com",
	})
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表（需要认证）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} User
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/users [get]
func listUsersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	})
}

func main() {
	r := gin.Default()

	// 公开接口
	r.POST("/api/v1/login", loginHandler)

	// 需要认证的接口
	api := r.Group("/api/v1")
	api.Use(AuthMiddleware())
	{
		api.GET("/me", getCurrentUserHandler)
		api.GET("/users", listUsersHandler)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8080")
}
