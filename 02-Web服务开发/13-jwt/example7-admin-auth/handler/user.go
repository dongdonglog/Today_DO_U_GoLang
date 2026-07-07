package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-book/jwt/example7-admin-auth/model"
	"github.com/go-book/jwt/example7-admin-auth/response"
	"github.com/go-book/jwt/example7-admin-auth/store"
)

type UserHandler struct {
	store *store.MemoryStore
}

func NewUserHandler(store *store.MemoryStore) *UserHandler {
	return &UserHandler{
		store: store,
	}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users := h.store.ListUsers()

	response.Success(c, gin.H{
		"users": users,
		"total": len(users),
	})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	user, err := h.store.GetUserByID(id)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, user)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Password, req.Role, req.Email)
	if err != nil {
		response.Conflict(c, "username already exists")
		return
	}

	response.Created(c, user)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	err = h.store.DeleteUser(id)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.NoContent(c)
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "unauthorized")
		return
	}

	user, err := h.store.GetUserByID(userID.(int))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, user)
}
