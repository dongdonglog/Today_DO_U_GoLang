package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-book/api-version/example5-admin-version/model"
	"github.com/go-book/api-version/example5-admin-version/service"
)

// UserHandler v1 用户处理器
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// ListUsers 获取用户列表
func (h *UserHandler) ListUsers(c *gin.Context) {
	users := h.service.List()
	c.JSON(http.StatusOK, toV1Response(users))
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserV1Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := h.service.Create(&model.CreateUserInput{
		Name:  req.Name,
		Email: req.Email,
	})
	c.JSON(http.StatusCreated, toV1Response([]*model.User{user})[0])
}

// 转换为 v1 响应格式（不含 phone）
func toV1Response(users []*model.User) []gin.H {
	result := make([]gin.H, len(users))
	for i, u := range users {
		result[i] = gin.H{
			"id":    u.ID,
			"name":  u.Name,
			"email": u.Email,
		}
	}
	return result
}
