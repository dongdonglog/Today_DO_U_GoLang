package handler

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-book/file-upload/example7-admin-upload/response"
	"github.com/go-book/file-upload/example7-admin-upload/storage"
)

type FileHandler struct {
	storage *storage.LocalStorage
}

func NewFileHandler(storage *storage.LocalStorage) *FileHandler {
	return &FileHandler{storage: storage}
}

// UploadFile 上传文件
func (h *FileHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}

	// 保存文件
	relativePath, err := h.storage.Save(file)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"filename": filepath.Base(relativePath),
		"path":     relativePath,
		"url":      "/api/v1/files/" + relativePath,
		"size":     file.Size,
	})
}

// DownloadFile 下载文件
func (h *FileHandler) DownloadFile(c *gin.Context) {
	filename := c.Param("filepath")
	filename = strings.TrimPrefix(filename, "/")

	if !storage.IsValidFilename(filename) {
		response.BadRequest(c, "invalid filename")
		return
	}

	fullPath, err := h.storage.Get(filename)
	if err != nil {
		response.NotFound(c, "file not found")
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(filename))
	c.Header("Content-Type", "application/octet-stream")
	c.File(fullPath)
}

// PreviewFile 预览文件
func (h *FileHandler) PreviewFile(c *gin.Context) {
	filename := c.Param("filepath")
	filename = strings.TrimPrefix(filename, "/")

	if !storage.IsValidFilename(filename) {
		response.BadRequest(c, "invalid filename")
		return
	}

	fullPath, err := h.storage.Get(filename)
	if err != nil {
		response.NotFound(c, "file not found")
		return
	}

	c.File(fullPath)
}

// DeleteFile 删除文件
func (h *FileHandler) DeleteFile(c *gin.Context) {
	filename := c.Param("filepath")
	filename = strings.TrimPrefix(filename, "/")

	if !storage.IsValidFilename(filename) {
		response.BadRequest(c, "invalid filename")
		return
	}

	if err := h.storage.Delete(filename); err != nil {
		response.NotFound(c, "file not found")
		return
	}

	response.NoContent(c)
}
