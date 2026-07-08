package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-book/file-upload/example7-admin-upload/response"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
	Role     string `json:"role"`
	Email    string `json:"email"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AccessClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// 模拟用户存储
var users = map[string]*User{
	"admin": {
		ID:       1,
		Username: "admin",
		Password: mustHashPassword("admin123"),
		Role:     "admin",
		Email:    "admin@example.com",
	},
}

type UserHandler struct {
	accessSecret  string
	refreshSecret string
	accessExpire  time.Duration
	refreshExpire time.Duration
}

func mustHashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hashedPassword)
}

func NewUserHandler(accessSecret, refreshSecret string, accessExpire, refreshExpire time.Duration) *UserHandler {
	return &UserHandler{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
	}
}

// Login 登录
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	user, exists := users[req.Username]
	if !exists {
		response.Unauthorized(c, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response.Unauthorized(c, "invalid credentials")
		return
	}

	// 生成 Token
	accessToken, err := h.generateAccessToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "failed to generate token")
		return
	}

	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		response.InternalError(c, "failed to generate token")
		return
	}

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// GetCurrentUser 获取当前用户
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	for _, user := range users {
		if user.ID == userID.(int) {
			response.Success(c, user)
			return
		}
	}

	response.NotFound(c, "user not found")
}

func (h *UserHandler) generateAccessToken(userID int, role string) (string, error) {
	claims := AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.accessExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "admin-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.accessSecret))
}

func (h *UserHandler) generateRefreshToken(userID int) (string, error) {
	claims := RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.refreshExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "admin-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.refreshSecret))
}
