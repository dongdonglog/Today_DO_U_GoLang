package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-book/openapi/example6-admin-docs/model"
	"github.com/go-book/openapi/example6-admin-docs/response"
)

// UserHandler 用户处理器
type UserHandler struct{}

// NewUserHandler 创建用户处理器
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表，支持分页
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} response.UserListResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	size := c.DefaultQuery("size", "10")

	c.JSON(http.StatusOK, response.UserListResponse{
		Code:    0,
		Message: "success",
		Data: []model.User{
			{ID: 1, Username: "admin", Email: "admin@example.com", Role: model.RoleAdmin, CreatedAt: "2024-01-15T10:30:00Z"},
			{ID: 2, Username: "alice", Email: "alice@example.com", Role: model.RoleUser, CreatedAt: "2024-01-16T11:30:00Z"},
		},
		Pagination: response.Pagination{
			Page:  func() int { p, _ := strconv.Atoi(page); return p }(),
			Size:  func() int { s, _ := strconv.Atoi(size); return s }(),
			Total: 2,
		},
	})
}

// GetCurrentUser 获取当前用户
// @Summary 获取当前用户
// @Description 获取当前登录用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.UserResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /api/v1/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	c.JSON(http.StatusOK, response.UserResponse{
		Code:    0,
		Message: "success",
		Data: model.User{
			ID:        1,
			Username:  "admin",
			Email:     "admin@example.com",
			Role:      model.RoleAdmin,
			CreatedAt: "2024-01-15T10:30:00Z",
		},
	})
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body model.CreateUserRequest true "用户信息"
// @Success 201 {object} response.UserResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response.UserResponse{
		Code:    0,
		Message: "user created",
		Data: model.User{
			ID:        3,
			Username:  req.Username,
			Email:     req.Email,
			Role:      req.Role,
			CreatedAt: "2024-01-17T12:30:00Z",
		},
	})
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param user body model.UpdateUserRequest true "用户信息"
// @Success 200 {object} response.UserResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: "invalid user id",
		})
		return
	}

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.UserResponse{
		Code:    0,
		Message: "user updated",
		Data: model.User{
			ID:        userID,
			Username:  "alice",
			Email:     req.Email,
			Role:      req.Role,
			UpdatedAt: "2024-01-17T13:30:00Z",
		},
	})
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 204 "删除成功"
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
