package main

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 允许的文件扩展名
var allowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".pdf":  true,
	".txt":  true,
}

// 允许的 MIME 类型
var allowedTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"application/pdf": true,
	"text/plain":      true,
}

// 清理文件名
func sanitizeFilename(filename string) string {
	// 1. 去除路径信息
	filename = filepath.Base(filename)

	// 2. 去除特殊字符
	filename = regexp.MustCompile(`[^\w\.\-]`).ReplaceAllString(filename, "_")

	// 3. 生成唯一文件名
	ext := filepath.Ext(filename)
	name := uuid.New().String() + ext

	return name
}

// 验证文件类型
func validateFileType(file *multipart.FileHeader) error {
	// 1. 检查扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		return fmt.Errorf("file extension not allowed: %s", ext)
	}

	// 2. 检查文件头（Magic Number）
	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil {
		return err
	}

	mimeType := http.DetectContentType(buffer)
	if !allowedTypes[mimeType] {
		return fmt.Errorf("file content not allowed: %s", mimeType)
	}

	return nil
}

// 验证文件大小
func validateFileSize(file *multipart.FileHeader, maxSize int64) error {
	if file.Size > maxSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)", file.Size, maxSize)
	}
	return nil
}

func main() {
	// 创建上传目录
	os.MkdirAll("./uploads", 0755)

	r := gin.Default()

	// 限制上传文件大小
	r.MaxMultipartMemory = 10 << 20 // 10 MB

	r.POST("/api/v1/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "file is required",
			})
			return
		}

		// 1. 验证文件大小
		if err := validateFileSize(file, 10<<20); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
			})
			return
		}

		// 2. 验证文件类型
		if err := validateFileType(file); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
			})
			return
		}

		// 3. 清理文件名
		safeFilename := sanitizeFilename(file.Filename)

		// 4. 保存文件
		if err := c.SaveUploadedFile(file, "./uploads/"+safeFilename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to save file",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":         0,
			"original":     file.Filename,
			"filename":     safeFilename,
			"size":         file.Size,
			"content_type": file.Header.Get("Content-Type"),
		})
	})

	fmt.Println("Server starting on :8080")
	fmt.Println("\n测试命令:")
	fmt.Println("  # 上传合法文件")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -F 'file=@test.jpg'")
	fmt.Println("\n  # 上传非法文件（会被拒绝）")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -F 'file=@test.exe'")
	fmt.Println("\n  # 上传过大文件（会被拒绝）")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -F 'file=@large.zip'")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
