package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// 创建上传目录
	os.MkdirAll("./uploads", 0755)

	r := gin.Default()

	// 限制上传文件大小（默认 32MB）
	r.MaxMultipartMemory = 8 << 20 // 8 MB

	// 单文件上传
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
		if err := c.SaveUploadedFile(file, "./uploads/"+file.Filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to save file",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":     0,
			"filename": file.Filename,
			"size":     file.Size,
		})
	})

	// 多文件上传
	r.POST("/api/v1/upload/multi", func(c *gin.Context) {
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
			if err := c.SaveUploadedFile(file, "./uploads/"+file.Filename); err != nil {
				continue
			}

			savedFiles = append(savedFiles, gin.H{
				"filename": file.Filename,
				"size":     file.Size,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code":  0,
			"files": savedFiles,
		})
	})

	fmt.Println("Server starting on :8080")
	fmt.Println("\n测试命令:")
	fmt.Println("  # 单文件上传")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -F 'file=@test.txt'")
	fmt.Println("\n  # 多文件上传")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload/multi \\")
	fmt.Println("    -F 'files=@file1.txt' \\")
	fmt.Println("    -F 'files=@file2.txt'")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
