package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-book/openapi/example6-admin-docs/response"
)

const uploadDir = "./uploads"

// FileHandler 文件处理器
type FileHandler struct{}

// NewFileHandler 创建文件处理器
func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

// UploadFile 上传文件
// @Summary 上传文件
// @Description 上传文件，支持图片和文档
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "文件"
// @Success 200 {object} response.FileResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /api/v1/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: "file is required",
		})
		return
	}

	filename, ok := safeFileName(filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/")))
	if !ok {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: "invalid filename",
		})
		return
	}

	c.JSON(http.StatusOK, response.FileResponse{
		Code:    0,
		Message: "file uploaded",
		Data: response.FileData{
			Filename: filename,
			URL:      "/api/v1/files/" + filename,
			Size:     file.Size,
		},
	})
}

// DownloadFile 下载文件
// @Summary 下载文件
// @Description 下载指定文件
// @Tags 文件管理
// @Produce application/octet-stream
// @Security BearerAuth
// @Param filename path string true "文件名"
// @Success 200 {file} binary
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/files/{filename} [get]
func (h *FileHandler) DownloadFile(c *gin.Context) {
	filename, ok := safeFileName(c.Param("filename"))
	if !ok {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: "invalid filename",
		})
		return
	}

	c.File(filepath.Join(uploadDir, filename))
}

// DeleteFile 删除文件
// @Summary 删除文件
// @Description 删除指定文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param filename path string true "文件名"
// @Success 204 "删除成功"
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/files/{filename} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	if _, ok := safeFileName(c.Param("filename")); !ok {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Code:    10001,
			Message: "invalid filename",
		})
		return
	}

	c.Status(http.StatusNoContent)
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
