package model

import "time"

// User 用户模型
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Phone        *string   `json:"phone,omitempty"`
	Status       int       `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Phone  *string `json:"phone"`
	Status *int    `json:"status"`
}

// ListUsersRequest 查询用户列表请求
type ListUsersRequest struct {
	Page   int    `form:"page" binding:"omitempty,gte=1"`
	Size   int    `form:"size" binding:"omitempty,gte=1,lte=100"`
	Email  string `form:"email"`
	Status *int   `form:"status"`
}
