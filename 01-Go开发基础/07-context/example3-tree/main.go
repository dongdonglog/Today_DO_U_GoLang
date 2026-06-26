package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 创建 Context 树
	root := context.Background()

	// 第一层：5 秒超时
	ctx1, cancel1 := context.WithTimeout(root, 5*time.Second)
	defer cancel1()

	// 第二层：传递 requestID
	ctx2 := context.WithValue(ctx1, "requestID", "req-123")

	// 第三层：手动取消
	ctx3, cancel3 := context.WithCancel(ctx2)
	defer cancel3()

	// 打印 Context 信息
	fmt.Println("Context 树结构：")
	fmt.Println("  Background")
	fmt.Println("    └── WithTimeout (5s)")
	fmt.Println("          └── WithValue (requestID=req-123)")
	fmt.Println("                └── WithCancel")

	fmt.Println("\n测试取消传播：")

	// 启动 goroutine 监听最内层 Context
	go func() {
		select {
		case <-ctx3.Done():
			fmt.Printf("ctx3 被取消: %v\n", ctx3.Err())
		}
	}()

	go func() {
		select {
		case <-ctx2.Done():
			fmt.Printf("ctx2 被取消: %v\n", ctx2.Err())
		}
	}()

	go func() {
		select {
		case <-ctx1.Done():
			fmt.Printf("ctx1 被取消: %v\n", ctx1.Err())
		}
	}()

	// 等待 2 秒，然后取消最内层
	time.Sleep(2 * time.Second)
	fmt.Println("\n取消 ctx3...")
	cancel3()

	time.Sleep(1 * time.Second)

	// 等待超时
	fmt.Println("\n等待 ctx1 超时...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n结论：父节点取消，子节点全部取消")
}
