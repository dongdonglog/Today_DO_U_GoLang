package main

import (
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

func main() {
	r := gin.Default()
	userService := NewUserService()

	// v1 路由组
	v1 := r.Group("/api/v1")
	{
		v1.GET("/users", func(c *gin.Context) {
			users := userService.List()
			c.JSON(200, toV1Response(users))
		})
	}

	// v2 路由组
	v2 := r.Group("/api/v2")
	{
		v2.GET("/users", func(c *gin.Context) {
			users := userService.List()
			c.JSON(200, toV2Response(users))
		})
	}

	r.Run(":8080")
}
