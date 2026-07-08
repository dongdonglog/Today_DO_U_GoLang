package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func sanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = regexp.MustCompile(`[^\w\.\-]`).ReplaceAllString(filename, "_")
	ext := filepath.Ext(filename)
	name := uuid.New().String() + ext
	return name
}

func isValidFilename(filename string) bool {
	// 防止路径遍历
	if strings.Contains(filename, "..") {
		return false
	}
	if strings.Contains(filename, "/") {
		return false
	}
	if strings.Contains(filename, "\\") {
		return false
	}
	return true
}

func main() {
	os.MkdirAll("./uploads", 0755)

	r := gin.Default()
	r.MaxMultipartMemory = 10 << 20

	// 上传文件
	r.POST("/api/v1/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "file is required",
			})
			return
		}

		safeFilename := sanitizeFilename(file.Filename)
		fullPath := filepath.Join("./uploads", safeFilename)

		if err := c.SaveUploadedFile(file, fullPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to save file",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":     0,
			"filename": safeFilename,
			"url":      "/api/v1/files/" + safeFilename,
			"size":     file.Size,
		})
	})

	// 下载文件
	r.GET("/api/v1/files/:filename", func(c *gin.Context) {
		filename := c.Param("filename")

		// 安全检查
		if !isValidFilename(filename) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "invalid filename",
			})
			return
		}

		filePath := filepath.Join("./uploads", filename)

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "file not found",
			})
			return
		}

		// 设置下载头
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "application/octet-stream")
		c.File(filePath)
	})

	// 预览文件（浏览器直接打开）
	r.GET("/api/v1/preview/:filename", func(c *gin.Context) {
		filename := c.Param("filename")

		if !isValidFilename(filename) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "invalid filename",
			})
			return
		}

		filePath := filepath.Join("./uploads", filename)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "file not found",
			})
			return
		}

		c.File(filePath)
	})

	fmt.Println("Server starting on :8080")
	fmt.Println("\n测试命令:")
	fmt.Println("  # 上传文件")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -F 'file=@test.jpg'")
	fmt.Println("\n  # 下载文件")
	fmt.Println("  curl http://localhost:8080/api/v1/files/abc-123.jpg -o downloaded.jpg")
	fmt.Println("\n  # 预览文件")
	fmt.Println("  open http://localhost:8080/api/v1/preview/abc-123.jpg")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
