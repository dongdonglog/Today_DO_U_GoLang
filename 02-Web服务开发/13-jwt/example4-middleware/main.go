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
var jwtSecret = []byte("your-secret-key")

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
	"alice": {
		ID:       2,
		Username: "alice",
		Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		Role:     "user",
	},
}

// JWT Claims
type UserClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 生成 Token
func generateToken(userID int, role string) (string, error) {
	claims := UserClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "my-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 解析 Token
func parseToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid authorization format",
			})
			c.Abort()
			return
		}

		// 验证 Token
		claims, err := parseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// 将用户信息存入 Context
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

	token, err := generateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"token": token,
		"user":  user,
	})
}

// 获取当前用户信息
func getCurrentUserHandler(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_id": userID,
			"role":    role,
		},
	})
}

// 受保护的用户列表接口
func listUsersHandler(c *gin.Context) {
	// 这里可以添加从数据库获取用户列表的逻辑
	users := []gin.H{
		{"id": 1, "username": "admin", "role": "admin"},
		{"id": 2, "username": "alice", "role": "user"},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": users,
	})
}

func main() {
	r := gin.Default()

	// 公开接口
	r.POST("/api/v1/login", loginHandler)

	// 需要认证的接口
	api := r.Group("/api/v1")
	api.Use(AuthMiddleware())
	{
		api.GET("/me", getCurrentUserHandler)
		api.GET("/users", listUsersHandler)
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("\n测试命令:")
	fmt.Println("  # 1. 登录获取 Token")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/login \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"admin\",\"password\":\"admin123\"}'")
	fmt.Println("\n  # 2. 访问受保护接口（需要 Token）")
	fmt.Println("  curl http://localhost:8080/api/v1/me \\")
	fmt.Println("    -H 'Authorization: Bearer <token>'")
	fmt.Println("\n  # 3. 访问用户列表（需要 Token）")
	fmt.Println("  curl http://localhost:8080/api/v1/users \\")
	fmt.Println("    -H 'Authorization: Bearer <token>'")
	fmt.Println("\n  # 4. 无 Token 访问（会被拒绝）")
	fmt.Println("  curl http://localhost:8080/api/v1/me")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
