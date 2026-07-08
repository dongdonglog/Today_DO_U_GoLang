package main

import (
	"github.com/gin-gonic/gin"
)

// UserV1 v1 版本用户模型（不含 phone）
type UserV1 struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserV2 v2 版本用户模型（包含 phone）
type UserV2 struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func main() {
	r := gin.Default()

	// v1 路由组
	v1 := r.Group("/api/v1")
	{
		v1.GET("/users", func(c *gin.Context) {
			c.JSON(200, []UserV1{
				{ID: 1, Name: "Alice", Email: "alice@example.com"},
				{ID: 2, Name: "Bob", Email: "bob@example.com"},
			})
		})
	}

	// v2 路由组
	v2 := r.Group("/api/v2")
	{
		v2.GET("/users", func(c *gin.Context) {
			c.JSON(200, []UserV2{
				{ID: 1, Name: "Alice", Email: "alice@example.com", Phone: "13800138000"},
				{ID: 2, Name: "Bob", Email: "bob@example.com", Phone: "13900139000"},
			})
		})
	}

	r.Run(":8080")
}
