package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ========================================
// 1. Reader/Writer 模式
// ========================================

// UpperCaseReader 大写转换器
type UpperCaseReader struct {
	r io.Reader
}

func (u *UpperCaseReader) Read(p []byte) (n int, err error) {
	n, err = u.r.Read(p)
	for i := 0; i < n; i++ {
		if p[i] >= 'a' && p[i] <= 'z' {
			p[i] -= 32 // 转大写
		}
	}
	return
}

// LowerCaseWriter 小写转换器
type LowerCaseWriter struct {
	w io.Writer
}

func (l *LowerCaseWriter) Write(p []byte) (n int, err error) {
	lower := make([]byte, len(p))
	for i, c := range p {
		if c >= 'A' && c <= 'Z' {
			lower[i] = c + 32
		} else {
			lower[i] = c
		}
	}
	return l.w.Write(lower)
}

func readerWriterPattern() {
	fmt.Println("=== Reader/Writer 模式 ===")

	// 读取并转大写
	src := strings.NewReader("hello world")
	upper := &UpperCaseReader{r: src}

	buf := make([]byte, 1024)
	n, _ := upper.Read(buf)
	fmt.Printf("Upper: %s\n", buf[:n])

	// 写入并转小写
	var dst bytes.Buffer
	lower := &LowerCaseWriter{w: &dst}
	lower.Write([]byte("HELLO WORLD"))
	fmt.Printf("Lower: %s\n", dst.String())
}

// ========================================
// 2. Handler 模式
// ========================================

// Handler 处理器接口
type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}

// ResponseWriter 响应写入器
type ResponseWriter interface {
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

// Request 请求
type Request struct {
	Method string
	Path   string
}

// HandlerFunc 函数适配器
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

// Middleware 中间件
type Middleware func(Handler) Handler

// LoggingMiddleware 日志中间件
func LoggingMiddleware(next Handler) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		fmt.Printf("[LOG] %s %s\n", r.Method, r.Path)
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware 认证中间件
func AuthMiddleware(next Handler) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		fmt.Println("[AUTH] Checking authentication")
		// 模拟认证失败
		if r.Path == "/admin" {
			w.WriteHeader(401)
			w.Write([]byte("Unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handlerPattern() {
	fmt.Println("\n=== Handler 模式 ===")

	// 创建处理器
	var handler Handler = HandlerFunc(func(w ResponseWriter, r *Request) {
		w.Write([]byte("Hello, World!"))
	})

	// 添加中间件
	handler = LoggingMiddleware(handler)
	handler = AuthMiddleware(handler)

	// 模拟请求
	req := Request{Method: "GET", Path: "/"}
	var resp mockResponseWriter
	handler.ServeHTTP(&resp, &req)
	fmt.Printf("Response: %s\n", resp.body)
}

type mockResponseWriter struct {
	body       string
	statusCode int
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.body += string(b)
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

// ========================================
// 3. Stringer 模式
// ========================================

// Point 点
type Point struct {
	X, Y int
}

// String 实现 fmt.Stringer
func (p Point) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}

// User 用户
type User struct {
	ID   int
	Name string
}

// String 实现 fmt.Stringer
func (u User) String() string {
	return fmt.Sprintf("User{ID: %d, Name: %s}", u.ID, u.Name)
}

func stringerPattern() {
	fmt.Println("\n=== Stringer 模式 ===")

	p := Point{X: 10, Y: 20}
	fmt.Printf("Point: %v\n", p)

	u := User{ID: 1, Name: "Alice"}
	fmt.Printf("User: %v\n", u)
}

// ========================================
// 4. Sort 模式
// ========================================

// Sortable 可排序接口
type Sortable interface {
	Len() int
	Less(i, j int) bool
	Swap(i, j int)
}

// IntSlice 整数切片
type IntSlice []int

func (s IntSlice) Len() int           { return len(s) }
func (s IntSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s IntSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Sort 排序
func Sort(data Sortable) {
	// 简单的冒泡排序
	n := data.Len()
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if data.Less(j+1, j) {
				data.Swap(j, j+1)
			}
		}
	}
}

func sortPattern() {
	fmt.Println("\n=== Sort 模式 ===")

	nums := IntSlice{5, 2, 8, 1, 9}
	fmt.Printf("Before: %v\n", nums)
	Sort(nums)
	fmt.Printf("After: %v\n", nums)
}

func main() {
	readerWriterPattern()
	handlerPattern()
	stringerPattern()
	sortPattern()
}
