package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	CodeSuccess      = 0
	CodeInternal     = 10000
	CodeInvalidParam = 10001
	CodeNotFound     = 10004

	CodeUserNotFound = 20001
	CodeUserExists   = 20002
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Pagination 分页信息
type Pagination struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

// ListResponse 列表响应（带分页）
type ListResponse struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors,omitempty"`
}

// FieldError 字段错误
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Created 创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    CodeSuccess,
		Message: "created",
		Data:    data,
	})
}

// NoContent 无内容响应（DELETE 成功）
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, data interface{}, total, page, size int) {
	c.JSON(http.StatusOK, ListResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
		Pagination: Pagination{
			Page:  page,
			Size:  size,
			Total: total,
		},
	})
}

// BadRequest 参数错误
func BadRequest(c *gin.Context, message string, errors []FieldError) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:    CodeInvalidParam,
		Message: message,
		Errors:  errors,
	})
}

// NotFound 资源不存在
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Code:    CodeUserNotFound,
		Message: message,
	})
}

// Conflict 资源冲突
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, ErrorResponse{
		Code:    CodeUserExists,
		Message: message,
	})
}

// InternalError 内部错误
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    CodeInternal,
		Message: message,
	})
}

// ValidationErrors 将 validator 的技术错误转换为字段级错误。
func ValidationErrors(err error) []FieldError {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return []FieldError{{Field: "request", Message: "invalid request body"}}
	}

	result := make([]FieldError, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		result = append(result, FieldError{
			Field:   fieldErr.Field(),
			Message: validationMessage(fieldErr),
		})
	}
	return result
}

func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fieldErr.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fieldErr.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fieldErr.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fieldErr.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fieldErr.Param())
	default:
		return "is invalid"
	}
}
