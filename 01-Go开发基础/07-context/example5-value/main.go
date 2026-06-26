package main

import (
	"context"
	"fmt"
)

// 类型安全的 key
type contextKey string

const (
	requestIDKey contextKey = "requestID"
	userIDKey    contextKey = "userID"
)

// 类型安全的封装
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(requestIDKey).(string)
	if !ok || id == "" {
		return "unknown"
	}
	return id
}

func WithUserID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserIDFromContext(ctx context.Context) int {
	id, ok := ctx.Value(userIDKey).(int)
	if !ok {
		return 0
	}
	return id
}

// 模拟中间件链
func middleware1(ctx context.Context) {
	fmt.Printf("中间件1: requestID=%s\n", RequestIDFromContext(ctx))

	// 添加 userID
	ctx = WithUserID(ctx, 123)
	middleware2(ctx)
}

func middleware2(ctx context.Context) {
	fmt.Printf("中间件2: requestID=%s, userID=%d\n",
		RequestIDFromContext(ctx),
		UserIDFromContext(ctx))

	handler(ctx)
}

func handler(ctx context.Context) {
	fmt.Printf("Handler: requestID=%s, userID=%d\n",
		RequestIDFromContext(ctx),
		UserIDFromContext(ctx))
}

func main() {
	fmt.Println("Context 值传递示例：")
	fmt.Println()

	// 创建根 Context
	ctx := context.Background()

	// 添加 requestID
	ctx = WithRequestID(ctx, "req-abc-123")

	// 进入中间件链
	middleware1(ctx)

	fmt.Println()
	fmt.Println("结论：Context 值沿着调用链向下传递")
	fmt.Println("每一层都可以添加新的值，下层可以读取上层的值")
}
