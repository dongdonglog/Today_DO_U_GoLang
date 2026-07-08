package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-book/api-version/example5-admin-version/model"
	"github.com/go-book/api-version/example5-admin-version/service"
)

// UserHandler v2 用户处理器
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
	c.JSON(http.StatusOK, toV2Response(users))
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := h.service.Create(&model.CreateUserInput{
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	})
	c.JSON(http.StatusCreated, toV2Response([]*model.User{user})[0])
}

// 转换为 v2 响应格式（包含 phone）
func toV2Response(users []*model.User) []gin.H {
	result := make([]gin.H, len(users))
	for i, u := range users {
		result[i] = gin.H{
			"id":    u.ID,
			"name":  u.Name,
			"email": u.Email,
			"phone": u.Phone,
		}
	}
	return result
}
