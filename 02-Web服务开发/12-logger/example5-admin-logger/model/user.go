package model

import "time"

// User 用户模型
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Name  string `json:"name" binding:"required,min=2,max=50"`
	Email string `json:"email" binding:"required,email"`
}

// UpdateUserReq 更新用户请求
type UpdateUserReq struct {
	Name  *string `json:"name,omitempty" binding:"omitempty,min=2,max=50"`
	Email *string `json:"email,omitempty" binding:"omitempty,email"`
}

// ListUsersReq 查询用户列表请求
type ListUsersReq struct {
	Page  int    `form:"page" binding:"omitempty,gte=1"`
	Size  int    `form:"size" binding:"omitempty,gte=1,lte=100"`
	Name  string `form:"name" binding:"omitempty"`
	Sort  string `form:"sort" binding:"omitempty"`
	Order string `form:"order" binding:"omitempty,oneof=asc desc"`
}
