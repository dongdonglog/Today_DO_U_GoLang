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

// 角色常量
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
	RoleGuest = "guest"
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
		Role:     RoleAdmin,
	},
	"alice": {
		ID:       2,
		Username: "alice",
		Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		Role:     RoleUser,
	},
	"guest": {
		ID:       3,
		Username: "guest",
		Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		Role:     RoleGuest,
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

		claims, err := parseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RBAC 权限中间件
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "unauthorized",
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "forbidden: insufficient permissions",
		})
		c.Abort()
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

// 公开接口：获取系统信息
func getSystemInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"version": "1.0.0",
			"status":  "running",
		},
	})
}

// 需要认证的接口：获取当前用户信息
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

// 需要 admin 权限的接口：删除用户
func deleteUserHandler(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("user %s deleted", userID),
	})
}

// 需要 admin 权限的接口：获取所有用户
func listAllUsersHandler(c *gin.Context) {
	users := []gin.H{
		{"id": 1, "username": "admin", "role": "admin"},
		{"id": 2, "username": "alice", "role": "user"},
		{"id": 3, "username": "guest", "role": "guest"},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": users,
	})
}

// 需要 user 或 admin 权限的接口：获取用户列表
func listUsersHandler(c *gin.Context) {
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
	r.GET("/api/v1/system/info", getSystemInfoHandler)

	// 需要认证的接口（所有角色）
	api := r.Group("/api/v1")
	api.Use(AuthMiddleware())
	{
		api.GET("/me", getCurrentUserHandler)
	}

	// 需要 user 或 admin 权限的接口
	userAPI := api.Group("")
	userAPI.Use(RequireRole(RoleAdmin, RoleUser))
	{
		userAPI.GET("/users", listUsersHandler)
	}

	// 需要 admin 权限的接口
	adminAPI := api.Group("/admin")
	adminAPI.Use(RequireRole(RoleAdmin))
	{
		adminAPI.GET("/users", listAllUsersHandler)
		adminAPI.DELETE("/users/:id", deleteUserHandler)
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("\n权限说明:")
	fmt.Println("  - admin: 所有权限")
	fmt.Println("  - user: 查看用户列表")
	fmt.Println("  - guest: 只能登录和查看系统信息")
	fmt.Println("\n测试命令:")
	fmt.Println("\n  # 1. 登录 admin")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/login \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"admin\",\"password\":\"admin123\"}'")
	fmt.Println("\n  # 2. 登录 user")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/login \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"alice\",\"password\":\"123456\"}'")
	fmt.Println("\n  # 3. admin 访问管理接口（成功）")
	fmt.Println("  curl http://localhost:8080/api/v1/admin/users \\")
	fmt.Println("    -H 'Authorization: Bearer <admin_token>'")
	fmt.Println("\n  # 4. user 访问管理接口（失败，403）")
	fmt.Println("  curl http://localhost:8080/api/v1/admin/users \\")
	fmt.Println("    -H 'Authorization: Bearer <user_token>'")
	fmt.Println("\n  # 5. 无 Token 访问（失败，401）")
	fmt.Println("  curl http://localhost:8080/api/v1/me")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
