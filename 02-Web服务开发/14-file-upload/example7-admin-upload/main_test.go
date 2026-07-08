package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-book/file-upload/example7-admin-upload/handler"
	"github.com/go-book/file-upload/example7-admin-upload/response"
	"github.com/go-book/file-upload/example7-admin-upload/storage"
)

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type loginData struct {
	AccessToken string `json:"access_token"`
}

type uploadData struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

func setupTestRouter(t *testing.T) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	fileStorage := storage.NewLocalStorage(t.TempDir(), []string{"image/jpeg"}, 1024*1024)
	userHandler := handler.NewUserHandler(strings.Repeat("a", 32), strings.Repeat("b", 32), 15*time.Minute, 7*24*time.Hour)
	fileHandler := handler.NewFileHandler(fileStorage)
	return newRouter(userHandler, fileHandler, strings.Repeat("a", 32))
}

func decodeAPIResponse(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func login(t *testing.T, h http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeAPIResponse(t, w)
	var data loginData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode login data: %v", err)
	}
	if data.AccessToken == "" {
		t.Fatalf("expected access token")
	}
	return data.AccessToken
}

func newJPEGUploadBody(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	if err := jpeg.Encode(part, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func TestUploadRequiresAuthentication(t *testing.T) {
	h := setupTestRouter(t)
	body, contentType := newJPEGUploadBody(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	resp := decodeAPIResponse(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected unauthorized code, got %d", resp.Code)
	}
}

func TestUploadThenPreview(t *testing.T) {
	h := setupTestRouter(t)
	token := login(t, h)
	body, contentType := newJPEGUploadBody(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected upload status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeAPIResponse(t, w)
	var data uploadData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode upload data: %v", err)
	}
	if data.Path == "" || strings.Contains(data.Path, "..") {
		t.Fatalf("unexpected upload path: %q", data.Path)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/"+data.Path, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected preview status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestPreviewRejectsTraversal(t *testing.T) {
	h := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/../secret.txt", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
