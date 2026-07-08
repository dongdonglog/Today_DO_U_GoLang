package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 上传信息
type UploadInfo struct {
	UploadID   string `json:"upload_id"`
	Filename   string `json:"filename"`
	FileSize   int64  `json:"file_size"`
	ChunkCount int    `json:"chunk_count"`
}

// 存储上传信息
var (
	uploads     = make(map[string]*UploadInfo)
	uploadsLock sync.RWMutex
)

const (
	uploadDir = "./uploads"
	tempDir   = "./temp"
)

func init() {
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(tempDir, 0755)
}

// 初始化上传
func initUploadHandler(c *gin.Context) {
	var req struct {
		Filename   string `json:"filename" binding:"required"`
		FileSize   int64  `json:"file_size" binding:"required"`
		ChunkCount int    `json:"chunk_count" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
		})
		return
	}

	uploadID := uuid.New().String()

	// 创建临时目录
	uploadTempDir := filepath.Join(tempDir, uploadID)
	os.MkdirAll(uploadTempDir, 0755)

	// 保存上传信息
	uploadsLock.Lock()
	uploads[uploadID] = &UploadInfo{
		UploadID:   uploadID,
		Filename:   req.Filename,
		FileSize:   req.FileSize,
		ChunkCount: req.ChunkCount,
	}
	uploadsLock.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"upload_id": uploadID,
	})
}

// 上传分片
func uploadChunkHandler(c *gin.Context) {
	uploadID := c.Param("upload_id")
	chunkIndex := c.Param("chunk_index")

	// 检查上传是否存在
	uploadsLock.RLock()
	_, exists := uploads[uploadID]
	uploadsLock.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "upload not found",
		})
		return
	}

	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "chunk is required",
		})
		return
	}

	// 保存分片
	chunkPath := filepath.Join(tempDir, uploadID, chunkIndex)
	if err := c.SaveUploadedFile(file, chunkPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to save chunk",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"chunk_index": chunkIndex,
		"message":     "chunk uploaded",
	})
}

// 合并分片
func mergeChunksHandler(c *gin.Context) {
	uploadID := c.Param("upload_id")

	// 获取上传信息
	uploadsLock.RLock()
	info, exists := uploads[uploadID]
	uploadsLock.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "upload not found",
		})
		return
	}

	// 合并分片
	finalPath := filepath.Join(uploadDir, info.Filename)
	uploadTempDir := filepath.Join(tempDir, uploadID)

	if err := mergeChunks(uploadTempDir, info.ChunkCount, finalPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to merge chunks",
		})
		return
	}

	// 清理临时文件
	os.RemoveAll(uploadTempDir)

	// 删除上传信息
	uploadsLock.Lock()
	delete(uploads, uploadID)
	uploadsLock.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"filename": info.Filename,
		"url":      "/api/v1/files/" + info.Filename,
	})
}

// 合并分片函数
func mergeChunks(tempDir string, chunkCount int, finalPath string) error {
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer finalFile.Close()

	for i := 0; i < chunkCount; i++ {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(finalFile, chunkFile)
		chunkFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	r := gin.Default()
	r.MaxMultipartMemory = 10 << 20

	// 分片上传接口
	r.POST("/api/v1/upload/init", initUploadHandler)
	r.POST("/api/v1/upload/:upload_id/chunk/:chunk_index", uploadChunkHandler)
	r.POST("/api/v1/upload/:upload_id/merge", mergeChunksHandler)

	// 文件访问
	r.Static("/api/v1/files", uploadDir)

	fmt.Println("Server starting on :8080")
	fmt.Println("\n分片上传流程:")
	fmt.Println("  1. 初始化上传")
	fmt.Println("  2. 上传分片")
	fmt.Println("  3. 合并分片")
	fmt.Println("\n测试命令:")
	fmt.Println("  # 1. 初始化上传")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload/init \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"filename\":\"large.zip\",\"file_size\":10485760,\"chunk_count\":2}'")
	fmt.Println("\n  # 2. 上传分片")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload/<upload_id>/chunk/0 \\")
	fmt.Println("    -F 'chunk=@chunk0'")
	fmt.Println("\n  # 3. 合并分片")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload/<upload_id>/merge")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
