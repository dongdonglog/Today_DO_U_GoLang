package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// 密钥
var (
	jwtAccessSecret  = []byte("your-access-secret-key")
	jwtRefreshSecret = []byte("your-refresh-secret-key")
)

// 用户模型
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
	Role     string `json:"role"`
}

// 模拟用户存储
var users = map[string]User{
	"admin": {
		ID:       1,
		Username: "admin",
		Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		Role:     "admin",
	},
}

// Access Token Claims
type AccessClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Refresh Token Claims
type RefreshClaims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Token 响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // 秒
}

// 刷新请求
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// 生成 Access Token
func generateAccessToken(userID int, role string) (string, error) {
	claims := AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // 15分钟
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "my-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtAccessSecret)
}

// 生成 Refresh Token
func generateRefreshToken(userID int) (string, error) {
	claims := RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7天
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "my-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtRefreshSecret)
}

// 解析 Access Token
func parseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtAccessSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AccessClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// 解析 Refresh Token
func parseRefreshToken(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtRefreshSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*RefreshClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Access Token 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid authorization format",
			})
			c.Abort()
			return
		}

		claims, err := parseAccessToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired access token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// 登录处理器
func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
		})
		return
	}

	user, exists := users[req.Username]
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "invalid credentials",
		})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "invalid credentials",
		})
		return
	}

	// 生成双 Token
	accessToken, err := generateAccessToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to generate access token",
		})
		return
	}

	refreshToken, err := generateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to generate refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15分钟 = 900秒
	})
}

// 刷新 Token 处理器
func refreshHandler(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
		})
		return
	}

	// 验证 Refresh Token
	claims, err := parseRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "invalid or expired refresh token",
		})
		return
	}

	// 查找用户
	var userRole string
	for _, user := range users {
		if user.ID == claims.UserID {
			userRole = user.Role
			break
		}
	}

	if userRole == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not found",
		})
		return
	}

	// 生成新的 Access Token
	newAccessToken, err := generateAccessToken(claims.UserID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to generate access token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"access_token": newAccessToken,
		"expires_in":   900,
	})
}

// 受保护的接口
func protectedHandler(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_id": userID,
			"role":    role,
			"message": "this is protected data",
		},
	})
}

func main() {
	r := gin.Default()

	// 公开接口
	r.POST("/api/v1/login", loginHandler)
	r.POST("/api/v1/refresh", refreshHandler)

	// 需要认证的接口
	api := r.Group("/api/v1")
	api.Use(AuthMiddleware())
	{
		api.GET("/protected", protectedHandler)
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("\nToken 说明:")
	fmt.Println("  - Access Token: 15分钟过期，用于访问资源")
	fmt.Println("  - Refresh Token: 7天过期，用于刷新 Access Token")
	fmt.Println("\n测试命令:")
	fmt.Println("\n  # 1. 登录获取双 Token")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/login \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"admin\",\"password\":\"admin123\"}'")
	fmt.Println("\n  # 2. 使用 Access Token 访问资源")
	fmt.Println("  curl http://localhost:8080/api/v1/protected \\")
	fmt.Println("    -H 'Authorization: Bearer <access_token>'")
	fmt.Println("\n  # 3. 使用 Refresh Token 刷新")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/refresh \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"refresh_token\":\"<refresh_token>\"}'")
	fmt.Println("\n  # 4. 使用新的 Access Token 访问资源")
	fmt.Println("  curl http://localhost:8080/api/v1/protected \\")
	fmt.Println("    -H 'Authorization: Bearer <new_access_token>'")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
