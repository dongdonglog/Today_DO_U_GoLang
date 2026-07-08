package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-book/openapi/example6-admin-docs/model"
	"github.com/go-book/openapi/example6-admin-docs/response"
)

// AuthHandler 认证处理器
type AuthHandler struct{}

// NewAuthHandler 创建认证处理器
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录，获取 Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body model.LoginRequest true "登录信息"
// @Success 200 {object} response.LoginResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /api/v1/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: err.Error(),
		})
		return
	}

	// 文档示例使用固定账号演示认证流程，真实项目应对接第 13 章的 JWT 签发逻辑。
	if req.Username == "admin" && req.Password == "admin123" {
		c.JSON(http.StatusOK, response.LoginResponse{
			Code:    0,
			Message: "success",
			Data: response.LoginData{
				AccessToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
				RefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			},
		})
		return
	}

	c.JSON(http.StatusUnauthorized, response.ErrorResponse{
		Code:    10002,
		Message: "invalid credentials",
	})
}

// AuthMiddleware 认证中间件
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, response.ErrorResponse{
				Code:    10002,
				Message: "missing token",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			c.JSON(http.StatusUnauthorized, response.ErrorResponse{
				Code:    10002,
				Message: "invalid token format",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
