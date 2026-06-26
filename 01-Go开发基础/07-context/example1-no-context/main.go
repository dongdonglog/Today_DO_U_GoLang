package main

import (
	"fmt"
	"runtime"
	"time"
)

// 模拟慢查询
func slowQuery(userID int) string {
	time.Sleep(10 * time.Second) // 模拟数据库慢查询
	return fmt.Sprintf("user-%d-data", userID)
}

// 没有 Context 的处理器
func handleRequest(userID int) {
	// 开始查询
	result := slowQuery(userID)
	fmt.Printf("查询完成: %s\n", result)
}

func main() {
	fmt.Println("启动服务...")

	// 模拟 100 个请求
	for i := 0; i < 100; i++ {
		go handleRequest(i)
	}

	// 每 2 秒打印一次 goroutine 数量
	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		fmt.Printf("[%ds] goroutine 数量: %d\n", (i+1)*2, runtime.NumGoroutine())
	}

	fmt.Println("\n问题：客户端早断了，但 goroutine 还在傻等")
	fmt.Println("100 个请求 = 100 个 goroutine 卡住 10 秒")
}
