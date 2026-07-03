package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-book/config/example5-admin-config/model"
	"github.com/go-book/config/example5-admin-config/response"
	"github.com/go-book/config/example5-admin-config/store"
)

// UserHandler 用户处理器
type UserHandler struct {
	store *store.MemoryStore
}

// NewUserHandler 创建用户处理器
func NewUserHandler(s *store.MemoryStore) *UserHandler {
	return &UserHandler{store: s}
}

// ListUsers 获取用户列表
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req model.ListUsersReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid query parameters", nil)
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}
	if req.Order == "" {
		req.Order = "asc"
	}

	users, total := h.store.List(req.Page, req.Size, req.Name, req.Sort, req.Order)
	response.SuccessWithPage(c, users, total, req.Page, req.Size)
}

// GetUser 获取用户详情
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	user, err := h.store.GetByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, user)
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := parseValidationErrors(err)
		response.BadRequest(c, "validation failed", errors)
		return
	}

	user, err := h.store.Create(req.Name, req.Email)
	if err != nil {
		response.Conflict(c, err.Error())
		return
	}

	response.Created(c, user)
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	var req model.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := parseValidationErrors(err)
		response.BadRequest(c, "validation failed", errors)
		return
	}

	user, err := h.store.Update(id, req.Name, req.Email)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, user)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	if err := h.store.Delete(id); err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.NoContent(c)
}

// parseValidationErrors 解析验证错误
func parseValidationErrors(err error) []response.FieldError {
	return []response.FieldError{
		{
			Field:   "request",
			Message: err.Error(),
		},
	}
}
