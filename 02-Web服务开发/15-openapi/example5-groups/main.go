package main

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Admin API
// @version 1.0
// @description 后台管理系统 API 文档
// @host localhost:8080
// @BasePath /

// User 用户模型
type User struct {
	ID    int    `json:"id" example:"1"`
	Name  string `json:"name" example:"Alice"`
	Email string `json:"email" example:"alice@example.com"`
}

// File 文件模型
type File struct {
	Filename string `json:"filename" example:"abc-123.jpg"`
	URL      string `json:"url" example:"/api/v1/files/abc-123.jpg"`
	Size     int64  `json:"size" example:"12345"`
}

// ===== 用户管理接口 =====

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表，支持分页
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {array} User
// @Router /api/v1/users [get]
func listUsersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	})
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param user body User true "用户信息"
// @Success 201 {object} User
// @Router /api/v1/users [post]
func createUserHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
}

// ===== 文件管理接口 =====

// UploadFile 上传文件
// @Summary 上传文件
// @Description 上传文件，支持图片和文档
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文件"
// @Success 200 {object} File
// @Router /api/v1/upload [post]
func uploadFileHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	filename, ok := safeFileName(filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/")))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	c.JSON(http.StatusOK, File{
		Filename: filename,
		URL:      "/api/v1/files/" + filename,
		Size:     file.Size,
	})
}

// DownloadFile 下载文件
// @Summary 下载文件
// @Description 下载指定文件
// @Tags 文件管理
// @Produce application/octet-stream
// @Param filename path string true "文件名"
// @Success 200 {file} binary
// @Router /api/v1/files/{filename} [get]
func downloadFileHandler(c *gin.Context) {
	filename, ok := safeFileName(c.Param("filename"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}
	c.File(filepath.Join("./uploads", filename))
}

// ===== 系统管理接口 =====

// HealthCheck 健康检查
// @Summary 健康检查
// @Description 检查服务是否正常运行
// @Tags 系统管理
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetSystemInfo 获取系统信息
// @Summary 获取系统信息
// @Description 获取系统版本、运行时间等信息
// @Tags 系统管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/system/info [get]
func getSystemInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":   "1.0.0",
		"uptime":    "24h",
		"goroutine": 50,
	})
}

func main() {
	r := gin.Default()

	// 系统管理
	r.GET("/health", healthCheckHandler)
	r.GET("/api/v1/system/info", getSystemInfoHandler)

	// 用户管理
	r.GET("/api/v1/users", listUsersHandler)
	r.POST("/api/v1/users", createUserHandler)

	// 文件管理
	r.POST("/api/v1/upload", uploadFileHandler)
	r.GET("/api/v1/files/:filename", downloadFileHandler)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8080")
}

func safeFileName(raw string) (string, bool) {
	if raw == "" || raw == "." || raw == ".." {
		return "", false
	}
	if strings.Contains(raw, "/") || strings.Contains(raw, "\\") {
		return "", false
	}
	cleaned := filepath.Clean(raw)
	if cleaned != raw {
		return "", false
	}
	return cleaned, true
}
