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

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, data interface{}, total, page, size int) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 参数错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, CodeInvalidParam, message)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, CodeUserNotFound, message)
}

// Conflict 资源冲突
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, CodeUserExists, message)
}

// InternalError 内部错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeInternal, message)
}

// ValidationMessage 将 validator 的技术错误转换为客户端可读文案。
func ValidationMessage(err error) string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return "invalid request body"
	}

	for _, fieldErr := range validationErrors {
		field := fieldErr.Field()
		switch fieldErr.Tag() {
		case "required":
			return fmt.Sprintf("%s is required", field)
		case "email":
			return fmt.Sprintf("%s must be a valid email", field)
		case "min":
			return fmt.Sprintf("%s must be at least %s characters", field, fieldErr.Param())
		case "max":
			return fmt.Sprintf("%s must be at most %s characters", field, fieldErr.Param())
		case "gte":
			return fmt.Sprintf("%s must be greater than or equal to %s", field, fieldErr.Param())
		case "lte":
			return fmt.Sprintf("%s must be less than or equal to %s", field, fieldErr.Param())
		default:
			return fmt.Sprintf("%s is invalid", field)
		}
	}

	return "invalid request body"
}
