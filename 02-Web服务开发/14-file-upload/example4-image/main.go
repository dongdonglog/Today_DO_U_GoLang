package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

// 生成缩略图
func generateThumbnail(src string, dst string, width, height uint) error {
	// 打开原图
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// 生成缩略图
	thumb := resize.Thumbnail(width, height, img, resize.Lanczos3)

	// 保存缩略图
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, thumb, nil)
}

// 压缩图片
func compressImage(src string, dst string, quality int) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
}

func sanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = regexp.MustCompile(`[^\w\.\-]`).ReplaceAllString(filename, "_")
	ext := filepath.Ext(filename)
	name := uuid.New().String() + ext
	return name
}

func main() {
	os.MkdirAll("./uploads", 0755)
	os.MkdirAll("./thumbnails", 0755)

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

		// 检查是否是图片
		ext := filepath.Ext(file.Filename)
		isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png"

		// 生成唯一文件名
		safeFilename := sanitizeFilename(file.Filename)
		fullPath := filepath.Join("./uploads", safeFilename)

		// 保存原图
		if err := c.SaveUploadedFile(file, fullPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to save file",
			})
			return
		}

		result := gin.H{
			"code":     0,
			"filename": safeFilename,
			"url":      "/api/v1/files/" + safeFilename,
			"size":     file.Size,
		}

		// 如果是图片，生成缩略图
		if isImage {
			thumbFilename := "thumb-" + safeFilename
			thumbPath := filepath.Join("./thumbnails", thumbFilename)

			if err := generateThumbnail(fullPath, thumbPath, 200, 200); err == nil {
				result["thumbnail"] = "/api/v1/thumbnails/" + thumbFilename
			}
		}

		c.JSON(http.StatusOK, result)
	})

	// 访问原图
	r.Static("/api/v1/files", "./uploads")
	// 访问缩略图
	r.Static("/api/v1/thumbnails", "./thumbnails")

	fmt.Println("Server starting on :8080")
	fmt.Println("\n测试命令:")
	fmt.Println("  # 上传图片")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -F 'file=@photo.jpg'")
	fmt.Println("\n  # 访问原图")
	fmt.Println("  curl http://localhost:8080/api/v1/files/abc-123.jpg")
	fmt.Println("\n  # 访问缩略图")
	fmt.Println("  curl http://localhost:8080/api/v1/thumbnails/thumb-abc-123.jpg")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
