package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type contextKey string

const requestIDKey contextKey = "requestID"

func requestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDKey).(string)
	if !ok || requestID == "" {
		return "unknown"
	}
	return requestID
}

// requestID 中间件
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// 日志中间件
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := requestIDFromContext(r.Context())

		next.ServeHTTP(w, r)

		log.Printf("[%s] %s %s %v",
			requestID,
			r.Method,
			r.URL.Path,
			time.Since(start))
	})
}

// 模拟慢查询
func slowQuery(ctx context.Context) (string, error) {
	select {
	case <-time.After(10 * time.Second):
		return "query result", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// 用户查询接口
func handleUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := requestIDFromContext(ctx)

	log.Printf("[%s] 开始查询用户...", requestID)

	result, err := slowQuery(ctx)
	if err != nil {
		log.Printf("[%s] 查询失败: %v", requestID, err)
		http.Error(w, "query failed", http.StatusGatewayTimeout)
		return
	}

	log.Printf("[%s] 查询成功: %s", requestID, result)
	w.Write([]byte(result))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", handleUser)

	// 应用中间件
	handler := requestIDMiddleware(mux)
	handler = loggingMiddleware(handler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	// 优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("收到关闭信号，开始优雅关闭...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("关闭失败: %v", err)
		}
	}()

	log.Println("服务启动在 :8080")
	log.Println("测试命令:")
	log.Println("  curl -H 'X-Request-ID: test-123' http://localhost:8080/api/user")
	log.Println("  # 按 Ctrl+C 优雅关闭")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("启动失败: %v", err)
	}

	log.Println("服务已关闭")
}
