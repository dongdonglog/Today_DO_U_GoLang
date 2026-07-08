package main

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 本地存储
type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	os.MkdirAll(basePath, 0755)
	return &LocalStorage{basePath: basePath}
}

// 生成存储路径（按日期分目录）
func (s *LocalStorage) generatePath() string {
	now := time.Now()
	datePath := now.Format("2006/01/02")
	fullPath := filepath.Join(s.basePath, datePath)
	os.MkdirAll(fullPath, 0755)
	return datePath
}

// 清理文件名
func sanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = regexp.MustCompile(`[^\w\.\-]`).ReplaceAllString(filename, "_")
	ext := filepath.Ext(filename)
	name := uuid.New().String() + ext
	return name
}

// 保存文件
func (s *LocalStorage) Save(c *gin.Context, file *multipart.FileHeader) (string, error) {
	// 生成日期目录
	datePath := s.generatePath()

	// 生成唯一文件名
	safeFilename := sanitizeFilename(file.Filename)

	// 完整路径
	relativePath := filepath.Join(datePath, safeFilename)
	fullPath := filepath.Join(s.basePath, relativePath)

	// 保存文件
	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		return "", err
	}

	return relativePath, nil
}

func main() {
	storage := NewLocalStorage("./uploads")

	r := gin.Default()
	r.MaxMultipartMemory = 10 << 20

	r.POST("/api/v1/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "file is required",
			})
			return
		}

		// 保存文件
		relativePath, err := storage.Save(c, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to save file",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":     0,
			"filename": filepath.Base(relativePath),
			"path":     relativePath,
			"url":      "/api/v1/files/" + relativePath,
			"size":     file.Size,
		})
	})

	// 文件访问
	r.Static("/api/v1/files", "./uploads")

	fmt.Println("Server starting on :8080")
	fmt.Println("\n测试命令:")
	fmt.Println("  # 上传文件")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -F 'file=@test.jpg'")
	fmt.Println("\n  # 访问文件")
	fmt.Println("  curl http://localhost:8080/api/v1/files/2024/01/15/abc-123.jpg")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
