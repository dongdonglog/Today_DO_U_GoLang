package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// 模拟慢查询，支持取消
func slowQuery(ctx context.Context, userID int) (string, error) {
	select {
	case <-time.After(10 * time.Second):
		return fmt.Sprintf("user-%d-data", userID), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// 使用 Context 的处理器
func handleRequest(ctx context.Context, userID int) {
	// 设置 3 秒超时
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := slowQuery(ctx, userID)
	if err != nil {
		fmt.Printf("请求 %d 失败: %v\n", userID, err)
		return
	}
	fmt.Printf("请求 %d 成功: %s\n", userID, result)
}

func main() {
	fmt.Println("启动服务（带超时控制）...")

	ctx := context.Background()

	// 模拟 100 个请求
	for i := 0; i < 100; i++ {
		go handleRequest(ctx, i)
	}

	// 每 2 秒打印一次 goroutine 数量
	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		fmt.Printf("[%ds] goroutine 数量: %d\n", (i+1)*2, runtime.NumGoroutine())
	}

	fmt.Println("\n改进：3 秒后自动取消，goroutine 数量快速下降")
}
