package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	uploadDir   = "uploads"
	maxFileSize = 8 << 20 // 8 MB
)

var allowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".pdf":  true,
	".txt":  true,
}

func validateUploadFile(filename string, size int64) (string, error) {
	if size <= 0 || size > maxFileSize {
		return "", fmt.Errorf("file size must be between 1 byte and 8 MB")
	}

	base := filepath.Base(filename)
	ext := strings.ToLower(filepath.Ext(base))
	if !allowedExts[ext] {
		return "", fmt.Errorf("unsupported file type")
	}
	return ext, nil
}

func newUploadName(ext string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	}
	return fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), hex.EncodeToString(buf), ext)
}

func main() {
	r := gin.Default()

	// 限制上传文件大小（默认 32MB）
	r.MaxMultipartMemory = maxFileSize
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(err)
	}

	// 上传页面
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
			<h2>单文件上传</h2>
			<form action="/upload" method="POST" enctype="multipart/form-data">
				<input type="file" name="file">
				<button type="submit">上传</button>
			</form>
			<h2>多文件上传</h2>
			<form action="/upload/multi" method="POST" enctype="multipart/form-data">
				<input type="file" name="files" multiple>
				<button type="submit">上传</button>
			</form>
		`)
	})

	// 单文件上传
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "file is required",
			})
			return
		}

		ext, err := validateUploadFile(file.Filename, file.Size)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
			})
			return
		}

		filename := newUploadName(ext)
		savePath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to save file",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "file uploaded",
			"data": gin.H{
				"filename": filepath.Base(file.Filename),
				"size":     file.Size,
				"saved_as": savePath,
			},
		})
	})

	// 多文件上传
	r.POST("/upload/multi", func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "invalid form data",
			})
			return
		}

		files := form.File["files"]
		savedFiles := make([]gin.H, 0, len(files))

		for _, file := range files {
			ext, err := validateUploadFile(file.Filename, file.Size)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": fmt.Sprintf("%s: %s", filepath.Base(file.Filename), err.Error()),
				})
				return
			}

			filename := newUploadName(ext)
			savePath := filepath.Join(uploadDir, filename)

			if err := c.SaveUploadedFile(file, savePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": fmt.Sprintf("failed to save %s", filepath.Base(file.Filename)),
				})
				return
			}

			savedFiles = append(savedFiles, gin.H{
				"filename": filepath.Base(file.Filename),
				"size":     file.Size,
				"saved_as": savePath,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "files uploaded",
			"data":    savedFiles,
		})
	})

	r.Run(":8080")
}
