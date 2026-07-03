package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-book/logger/example5-admin-logger/model"
	"github.com/go-book/logger/example5-admin-logger/response"
	"github.com/go-book/logger/example5-admin-logger/store"
	"go.uber.org/zap"
)

// UserHandler 用户处理器
type UserHandler struct {
	store  *store.MemoryStore
	logger *zap.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(s *store.MemoryStore, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		store:  s,
		logger: logger,
	}
}

// ListUsers 获取用户列表
func (h *UserHandler) ListUsers(c *gin.Context) {
	requestID := c.GetString("request_id")

	var req model.ListUsersReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warn("invalid query parameters",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
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

	h.logger.Info("user list retrieved",
		zap.String("request_id", requestID),
		zap.Int("count", len(users)),
		zap.Int("total", total),
	)

	response.SuccessWithPage(c, users, total, req.Page, req.Size)
}

// GetUser 获取用户详情
func (h *UserHandler) GetUser(c *gin.Context) {
	requestID := c.GetString("request_id")

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("invalid user id",
			zap.String("request_id", requestID),
			zap.String("id", c.Param("id")),
		)
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	user, err := h.store.GetByID(id)
	if err != nil {
		h.logger.Warn("user not found",
			zap.String("request_id", requestID),
			zap.Int("user_id", id),
		)
		response.NotFound(c, err.Error())
		return
	}

	h.logger.Info("user retrieved",
		zap.String("request_id", requestID),
		zap.Int("user_id", id),
	)

	response.Success(c, user)
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	requestID := c.GetString("request_id")

	var req model.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid request body",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		errors := parseValidationErrors(err)
		response.BadRequest(c, "validation failed", errors)
		return
	}

	user, err := h.store.Create(req.Name, req.Email)
	if err != nil {
		h.logger.Warn("create user failed",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		response.Conflict(c, err.Error())
		return
	}

	h.logger.Info("user created",
		zap.String("request_id", requestID),
		zap.Int("user_id", user.ID),
		zap.String("name", user.Name),
		zap.String("email", user.Email),
	)

	response.Created(c, user)
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	requestID := c.GetString("request_id")

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("invalid user id",
			zap.String("request_id", requestID),
			zap.String("id", c.Param("id")),
		)
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	var req model.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid request body",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		errors := parseValidationErrors(err)
		response.BadRequest(c, "validation failed", errors)
		return
	}

	user, err := h.store.Update(id, req.Name, req.Email)
	if err != nil {
		h.logger.Warn("update user failed",
			zap.String("request_id", requestID),
			zap.Int("user_id", id),
			zap.Error(err),
		)
		response.NotFound(c, err.Error())
		return
	}

	h.logger.Info("user updated",
		zap.String("request_id", requestID),
		zap.Int("user_id", id),
	)

	response.Success(c, user)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	requestID := c.GetString("request_id")

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn("invalid user id",
			zap.String("request_id", requestID),
			zap.String("id", c.Param("id")),
		)
		response.BadRequest(c, "invalid user id", nil)
		return
	}

	if err := h.store.Delete(id); err != nil {
		h.logger.Warn("delete user failed",
			zap.String("request_id", requestID),
			zap.Int("user_id", id),
			zap.Error(err),
		)
		response.NotFound(c, err.Error())
		return
	}

	h.logger.Info("user deleted",
		zap.String("request_id", requestID),
		zap.Int("user_id", id),
	)

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
