package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileStorage 文件存储接口
type FileStorage interface {
	Save(file *multipart.FileHeader) (string, error)
	Get(filename string) (string, error)
	Delete(filename string) error
}

// LocalStorage 本地存储
type LocalStorage struct {
	basePath     string
	allowedTypes map[string]bool
	maxSize      int64
}

// NewLocalStorage 创建本地存储
func NewLocalStorage(basePath string, allowedTypes []string, maxSize int64) *LocalStorage {
	_ = os.MkdirAll(basePath, 0755)

	types := make(map[string]bool)
	for _, t := range allowedTypes {
		types[t] = true
	}

	return &LocalStorage{
		basePath:     basePath,
		allowedTypes: types,
		maxSize:      maxSize,
	}
}

// Save 保存文件
func (s *LocalStorage) Save(file *multipart.FileHeader) (string, error) {
	// 1. 验证文件大小
	if file.Size > s.maxSize {
		return "", fmt.Errorf("file too large: %d bytes (max: %d bytes)", file.Size, s.maxSize)
	}

	// 2. 验证文件类型
	if err := s.validateFileType(file); err != nil {
		return "", err
	}

	// 3. 生成存储路径
	datePath := s.generateDatePath()
	fullDir := filepath.Join(s.basePath, datePath)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// 4. 生成唯一文件名
	safeFilename := s.sanitizeFilename(file.Filename)
	relativePath := filepath.Join(datePath, safeFilename)
	fullPath, err := s.safePath(relativePath)
	if err != nil {
		return "", err
	}

	// 5. 保存文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return "", err
	}

	return relativePath, nil
}

// Get 获取文件路径
func (s *LocalStorage) Get(filename string) (string, error) {
	fullPath, err := s.safePath(filename)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", filename)
	}

	return fullPath, nil
}

// Delete 删除文件
func (s *LocalStorage) Delete(filename string) error {
	fullPath, err := s.safePath(filename)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}

// validateFileType 验证文件类型
func (s *LocalStorage) validateFileType(file *multipart.FileHeader) error {
	// 检查文件头（Magic Number）
	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}
	if n == 0 {
		return fmt.Errorf("empty file")
	}

	mimeType := http.DetectContentType(buffer[:n])
	if !s.allowedTypes[mimeType] {
		return fmt.Errorf("file type not allowed: %s", mimeType)
	}

	return nil
}

// generateDatePath 生成日期路径
func (s *LocalStorage) generateDatePath() string {
	now := time.Now()
	return now.Format("2006/01/02")
}

// sanitizeFilename 清理文件名
func (s *LocalStorage) sanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = regexp.MustCompile(`[^\w\.\-]`).ReplaceAllString(filename, "_")

	ext := filepath.Ext(filename)
	name := uuid.New().String() + ext

	return name
}

// isValidFilename 检查文件名是否合法
func IsValidFilename(filename string) bool {
	clean := filepath.Clean(filename)
	if clean == "." || filepath.IsAbs(clean) {
		return false
	}
	if strings.HasPrefix(clean, "..") {
		return false
	}
	if strings.Contains(clean, "\\") {
		return false
	}
	return true
}

func (s *LocalStorage) safePath(filename string) (string, error) {
	if !IsValidFilename(filename) {
		return "", fmt.Errorf("invalid filename")
	}

	baseAbs, err := filepath.Abs(s.basePath)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(baseAbs, filepath.Clean(filename))
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(baseAbs, fullAbs)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid path")
	}

	return fullAbs, nil
}
