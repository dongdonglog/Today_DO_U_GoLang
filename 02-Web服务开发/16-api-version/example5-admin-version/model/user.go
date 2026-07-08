package model

// User 用户模型
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// CreateUserV1Request v1 创建用户请求
type CreateUserV1Request struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

// CreateUserV2Request v2 创建用户请求
type CreateUserV2Request struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Phone string `json:"phone" binding:"required"`
}

// CreateUserInput Service 层统一创建用户输入
type CreateUserInput struct {
	Name  string
	Email string
	Phone string
}
