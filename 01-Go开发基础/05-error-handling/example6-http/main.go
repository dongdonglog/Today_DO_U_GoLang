package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// ========================================
// 1. 错误码定义
// ========================================

const (
	CodeSuccess      = 0
	CodeInternal     = 10000
	CodeInvalidParam = 10001
	CodeUnauthorized = 10002
	CodeNotFound     = 10004
	CodeUserNotFound = 20001
)

// ========================================
// 2. 统一响应结构
// ========================================

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(data interface{}) APIResponse {
	return APIResponse{Code: CodeSuccess, Message: "success", Data: data}
}

func ErrorResp(code int, message string) APIResponse {
	return APIResponse{Code: code, Message: message}
}

// ========================================
// 3. 业务错误类型
// ========================================

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// ========================================
// 4. 用户数据
// ========================================

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = map[string]*User{
	"1": {ID: "1", Name: "Alice", Email: "alice@example.com"},
	"2": {ID: "2", Name: "Bob", Email: "bob@example.com"},
}

// ========================================
// 5. 业务逻辑
// ========================================

func getUser(id string) (*User, error) {
	if id == "" {
		return nil, &AppError{
			Code:    CodeInvalidParam,
			Message: "user id is required",
		}
	}

	user, ok := users[id]
	if !ok {
		return nil, &AppError{
			Code:    CodeUserNotFound,
			Message: fmt.Sprintf("user %s not found", id),
		}
	}

	return user, nil
}

func createUser(name, email string) (*User, error) {
	if name == "" || email == "" {
		return nil, &AppError{
			Code:    CodeInvalidParam,
			Message: "name and email are required",
		}
	}

	user := &User{
		ID:    fmt.Sprintf("%d", len(users)+1),
		Name:  name,
		Email: email,
	}
	users[user.ID] = user

	return user, nil
}

// ========================================
// 6. HTTP 处理器
// ========================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// handleGetUser 获取用户
func handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	user, err := getUser(id)
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			// 根据错误码返回不同的 HTTP 状态码
			status := http.StatusBadRequest
			if appErr.Code == CodeUserNotFound {
				status = http.StatusNotFound
			}
			writeJSON(w, status, appErr.ToResponse())
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResp(CodeInternal, "internal error"))
		return
	}

	writeJSON(w, http.StatusOK, Success(user))
}

// ToResponse 转换为 API 响应
func (e *AppError) ToResponse() APIResponse {
	return ErrorResp(e.Code, e.Message)
}

// handleCreateUser 创建用户
func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResp(CodeInvalidParam, "method not allowed"))
		return
	}

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResp(CodeInvalidParam, "invalid request body"))
		return
	}

	user, err := createUser(req.Name, req.Email)
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			writeJSON(w, http.StatusBadRequest, appErr.ToResponse())
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResp(CodeInternal, "internal error"))
		return
	}

	writeJSON(w, http.StatusCreated, Success(user))
}

// handlePanic 模拟 panic
func handlePanic(w http.ResponseWriter, r *http.Request) {
	// 使用 recover 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic recovered: %v", r)
			writeJSON(w, http.StatusInternalServerError, ErrorResp(CodeInternal, "internal server error"))
		}
	}()

	// 模拟 panic
	panic("something went wrong")
}

// recoveryMiddleware 恢复中间件
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v, stack: %s", err, debug.Stack())
				writeJSON(w, http.StatusInternalServerError, ErrorResp(CodeInternal, "internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ========================================
// 7. 主函数
// ========================================

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/user", handleGetUser)
	mux.HandleFunc("POST /api/user", handleCreateUser)
	mux.HandleFunc("GET /api/panic", handlePanic)

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("")
	fmt.Println("API Endpoints:")
	fmt.Println("  GET  /api/user?id=1    - Get user")
	fmt.Println("  GET  /api/user?id=999  - User not found")
	fmt.Println("  GET  /api/user         - Invalid param")
	fmt.Println("  POST /api/user         - Create user")
	fmt.Println("  GET  /api/panic        - Simulate panic")
	fmt.Println("")
	fmt.Println("Test commands:")
	fmt.Println("  curl 'http://localhost:8080/api/user?id=1'")
	fmt.Println("  curl 'http://localhost:8080/api/user?id=999'")
	fmt.Println("  curl 'http://localhost:8080/api/user'")
	fmt.Println("  curl -X POST http://localhost:8080/api/user \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"name\":\"Charlie\",\"email\":\"charlie@example.com\"}'")
	fmt.Println("  curl 'http://localhost:8080/api/panic'")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
