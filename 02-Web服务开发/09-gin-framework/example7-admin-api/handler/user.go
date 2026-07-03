package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-book/gin-framework/example7-admin-api/model"
	"github.com/go-book/gin-framework/example7-admin-api/response"
	"github.com/go-book/gin-framework/example7-admin-api/store"
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
		response.BadRequest(c, response.ValidationMessage(err))
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	users, total := h.store.List(req.Page, req.Size)
	response.SuccessWithPage(c, users, total, req.Page, req.Size)
}

// GetUser 获取用户详情
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id")
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
		response.BadRequest(c, response.ValidationMessage(err))
		return
	}

	user, err := h.store.Create(req.Name, req.Email)
	if err != nil {
		if errors.Is(err, store.ErrUserExists) {
			response.Conflict(c, "email already exists")
			return
		}
		response.InternalError(c, "failed to create user")
		return
	}

	response.Success(c, user)
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req model.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.ValidationMessage(err))
		return
	}

	user, err := h.store.Update(id, req.Name, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrUserNotFound):
			response.NotFound(c, "user not found")
		case errors.Is(err, store.ErrUserExists):
			response.Conflict(c, "email already exists")
		default:
			response.InternalError(c, "failed to update user")
		}
		return
	}

	response.Success(c, user)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.store.Delete(id); err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, nil)
}
