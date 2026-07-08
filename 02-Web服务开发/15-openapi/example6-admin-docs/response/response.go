package response

import "github.com/go-book/openapi/example6-admin-docs/model"

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code" example:"0"`
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// EmptyResponse 无响应体数据的统一响应结构
type EmptyResponse struct {
	Code    int    `json:"code" example:"0"`
	Message string `json:"message" example:"success"`
}

// Pagination 分页信息
type Pagination struct {
	Page  int `json:"page" example:"1"`
	Size  int `json:"size" example:"10"`
	Total int `json:"total" example:"100"`
}

// ListResponse 列表响应
type ListResponse struct {
	Code       int         `json:"code" example:"0"`
	Message    string      `json:"message" example:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// UserResponse 用户详情响应
type UserResponse struct {
	Code    int        `json:"code" example:"0"`
	Message string     `json:"message" example:"success"`
	Data    model.User `json:"data"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Code       int          `json:"code" example:"0"`
	Message    string       `json:"message" example:"success"`
	Data       []model.User `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// FileResponse 文件响应
type FileResponse struct {
	Code    int      `json:"code" example:"0"`
	Message string   `json:"message" example:"file uploaded"`
	Data    FileData `json:"data"`
}

// FileData 文件信息
type FileData struct {
	Filename string `json:"filename" example:"abc-123.jpg"`
	URL      string `json:"url" example:"/api/v1/files/abc-123.jpg"`
	Size     int64  `json:"size" example:"12345"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code" example:"10001"`
	Message string `json:"message" example:"invalid request"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Code    int       `json:"code" example:"0"`
	Message string    `json:"message" example:"success"`
	Data    LoginData `json:"data"`
}

// LoginData 登录成功后的 Token 数据
type LoginData struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
