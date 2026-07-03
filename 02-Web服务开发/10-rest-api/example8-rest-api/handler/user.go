package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-book/rest-api/example8-rest-api/model"
	"github.com/go-book/rest-api/example8-rest-api/response"
	"github.com/go-book/rest-api/example8-rest-api/store"
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
// GET /api/v1/users?page=1&size=10&name=Alice&sort=created_at&order=desc
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req model.ListUsersReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "validation failed", response.ValidationErrors(err))
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
// GET /api/v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	user, err := h.store.GetByID(id)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, user)
}

// CreateUser 创建用户
// POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "validation failed", response.ValidationErrors(err))
		return
	}

	user, err := h.store.CreateWithIdempotencyKey(c.GetHeader("Idempotency-Key"), req.Name, req.Email)
	if err != nil {
		if errors.Is(err, store.ErrUserExists) {
			response.Conflict(c, "email already exists")
			return
		}
		response.InternalError(c, "failed to create user")
		return
	}

	response.Created(c, user)
}

// UpdateUser 全量更新用户
// PUT /api/v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	var req model.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "validation failed", response.ValidationErrors(err))
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

// PatchUser 部分更新用户
// PATCH /api/v1/users/:id
func (h *UserHandler) PatchUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	var req model.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "validation failed", response.ValidationErrors(err))
		return
	}

	user, err := h.store.Patch(id, req.Name, req.Email)
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
// DELETE /api/v1/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	if err := h.store.Delete(id); err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.NoContent(c)
}
