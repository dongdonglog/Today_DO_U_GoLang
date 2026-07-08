package model

// UserRole 用户角色
// @Description 用户角色：admin 为管理员，user 为普通用户，guest 为访客
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
	Username  string   `json:"username" example:"alice" binding:"required,min=3,max=50"`
	Email     string   `json:"email" example:"alice@example.com" binding:"required,email"`
	Role      UserRole `json:"role" example:"user" enums:"admin,user,guest"`
	CreatedAt string   `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt string   `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string   `json:"username" example:"alice" binding:"required,min=3,max=50"`
	Email    string   `json:"email" example:"alice@example.com" binding:"required,email"`
	Password string   `json:"password" example:"ChangeMe_123" binding:"required,min=8"`
	Role     UserRole `json:"role" example:"user" enums:"admin,user,guest"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email string   `json:"email" example:"alice@example.com" binding:"omitempty,email"`
	Role  UserRole `json:"role" example:"user" enums:"admin,user,guest"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" example:"admin" binding:"required"`
	Password string `json:"password" example:"admin123" binding:"required"`
}
