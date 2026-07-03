package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Name  string `json:"name" binding:"required,min=2,max=50"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age" binding:"gte=0,lte=150"`
}

// UpdateUserReq 更新用户请求
type UpdateUserReq struct {
	Name  *string `json:"name,omitempty" binding:"omitempty,min=2,max=50"`
	Email *string `json:"email,omitempty" binding:"omitempty,email"`
}

// ListUsersReq 查询用户列表请求
type ListUsersReq struct {
	Page int `form:"page" binding:"gte=1"`
	Size int `form:"size" binding:"gte=1,lte=100"`
}

func main() {
	r := gin.Default()

	// JSON 绑定 + 验证
	r.POST("/users", func(c *gin.Context) {
		var req CreateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"code":    0,
			"message": "user created",
			"data": gin.H{
				"name":  req.Name,
				"email": req.Email,
				"age":   req.Age,
			},
		})
	})

	// 更新用户（部分字段）
	r.PUT("/users/:id", func(c *gin.Context) {
		id := c.Param("id")

		var req UpdateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "user updated",
			"data": gin.H{
				"id":    id,
				"name":  req.Name,
				"email": req.Email,
			},
		})
	})

	// 查询参数绑定
	r.GET("/users", func(c *gin.Context) {
		var req ListUsersReq
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"page": req.Page,
				"size": req.Size,
			},
		})
	})

	r.Run(":8080")
}
