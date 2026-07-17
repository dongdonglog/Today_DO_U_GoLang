package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-book/mysql/example2-admin-mysql/model"
	"github.com/go-book/mysql/example2-admin-mysql/store"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler 用户处理器
type UserHandler struct {
	store *store.MySQLStore
}

// NewUserHandler 创建用户处理器
func NewUserHandler(store *store.MySQLStore) *UserHandler {
	return &UserHandler{store: store}
}

// ListUsers 查询用户列表
// @Summary 查询用户列表
// @Description 查询用户列表，支持分页、过滤
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param email query string false "邮箱过滤"
// @Param status query int false "状态过滤"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req model.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": err.Error(),
		})
		return
	}

	users, total, err := h.store.List(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    10000,
			"message": err.Error(),
		})
		return
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.Size
	if size <= 0 {
		size = 10
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    users,
		"pagination": gin.H{
			"page":  page,
			"size":  size,
			"total": total,
		},
	})
}

// GetUser 查询用户详情
// @Summary 查询用户详情
// @Description 根据 ID 查询用户详情
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} model.User
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": "invalid user id",
		})
		return
	}

	user, err := h.store.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    10004,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    user,
	})
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param user body model.CreateUserRequest true "用户信息"
// @Success 201 {object} model.User
// @Failure 400 {object} map[string]string
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": err.Error(),
		})
		return
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    10000,
			"message": "failed to hash password",
		})
		return
	}

	user, err := h.store.Create(c.Request.Context(), &req, string(hashedPasswordBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "user created",
		"data":    user,
	})
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param user body model.UpdateUserRequest true "用户信息"
// @Success 200 {object} model.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": "invalid user id",
		})
		return
	}

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": err.Error(),
		})
		return
	}

	user, err := h.store.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    10004,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "user updated",
		"data":    user,
	})
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除用户（软删除）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 204 "删除成功"
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": "invalid user id",
		})
		return
	}

	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    10004,
			"message": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
