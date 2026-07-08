package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// User 统一的用户模型
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// UserService 共享的业务逻辑
type UserService struct {
	users []*User
}

func NewUserService() *UserService {
	return &UserService{
		users: []*User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Phone: "13800138000"},
			{ID: 2, Name: "Bob", Email: "bob@example.com", Phone: "13900139000"},
		},
	}
}

func (s *UserService) List() []*User {
	return s.users
}

// 响应格式转换
func toV1Response(users []*User) []gin.H {
	result := make([]gin.H, len(users))
	for i, u := range users {
		result[i] = gin.H{
			"id":    u.ID,
			"name":  u.Name,
			"email": u.Email,
		}
	}
	return result
}

func toV2Response(users []*User) []gin.H {
	result := make([]gin.H, len(users))
	for i, u := range users {
		result[i] = gin.H{
			"id":    u.ID,
			"name":  u.Name,
			"email": u.Email,
			"phone": u.Phone,
		}
	}
	return result
}

// 版本中间件
func VersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 URL 获取版本
		version := c.Param("version")

		// 2. 从 Header 获取版本
		if version == "" {
			version = c.GetHeader("X-API-Version")
		}

		// 3. 默认 v1
		if version == "" {
			version = "v1"
		}
		version = strings.TrimPrefix(strings.ToLower(version), "api-")
		if version == "1" || version == "2" {
			version = "v" + version
		}

		c.Set("api_version", version)
		c.Next()
	}
}

func listUsers(c *gin.Context, userService *UserService) {
	version := c.GetString("api_version")
	users := userService.List()

	switch version {
	case "v1":
		c.JSON(http.StatusOK, toV1Response(users))
	case "v2":
		c.JSON(http.StatusOK, toV2Response(users))
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": "unsupported api version",
		})
	}
}

func main() {
	r := gin.Default()
	userService := NewUserService()

	// 使用版本中间件
	r.Use(VersionMiddleware())

	// URL 版本：/api/v1/users、/api/v2/users
	r.GET("/api/:version/users", func(c *gin.Context) {
		listUsers(c, userService)
	})

	// Header 版本：/api/users + X-API-Version: v2
	r.GET("/api/users", func(c *gin.Context) {
		listUsers(c, userService)
	})

	r.Run(":8080")
}
