package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func main() {
	// ========================================
	// 1. error 接口
	// ========================================
	fmt.Println("=== error 接口 ===")

	// error 是一个接口
	// type error interface {
	//     Error() string
	// }

	// 创建错误的方式
	err1 := errors.New("something went wrong")
	fmt.Printf("errors.New: %v\n", err1)

	err2 := fmt.Errorf("failed to process user %d", 123)
	fmt.Printf("fmt.Errorf: %v\n", err2)

	// ========================================
	// 2. 基本错误处理
	// ========================================
	fmt.Println("\n=== 基本错误处理 ===")

	// 方式 1：直接返回
	if err := doSomething(); err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	// 方式 2：错误包装
	if err := doAnotherThing(); err != nil {
		fmt.Printf("包装后: %v\n", err)
	}

	// ========================================
	// 3. 文件操作错误处理
	// ========================================
	fmt.Println("\n=== 文件操作错误处理 ===")

	// 打开不存在的文件
	file, err := os.Open("nonexistent.txt")
	if err != nil {
		fmt.Printf("打开文件失败: %v\n", err)
	} else {
		file.Close()
	}

	// ========================================
	// 4. io.EOF 特殊错误
	// ========================================
	fmt.Println("\n=== io.EOF ===")

	// io.EOF 是一个哨兵错误
	err = io.EOF
	if err == io.EOF {
		fmt.Println("到达文件末尾")
	}

	// ========================================
	// 5. 错误处理模式
	// ========================================
	fmt.Println("\n=== 错误处理模式 ===")

	// 模式 1：失败快速返回
	result, err := processWithFastFail("input")
	if err != nil {
		fmt.Printf("快速失败: %v\n", err)
	} else {
		fmt.Printf("结果: %s\n", result)
	}

	// 模式 2：重试
	result, err = processWithRetry("input", 3)
	if err != nil {
		fmt.Printf("重试失败: %v\n", err)
	} else {
		fmt.Printf("重试成功: %s\n", result)
	}
}

func doSomething() error {
	return errors.New("something failed")
}

func doAnotherThing() error {
	err := doSomething()
	if err != nil {
		return fmt.Errorf("doAnotherThing failed: %w", err)
	}
	return nil
}

func processWithFastFail(input string) (string, error) {
	if input == "" {
		return "", errors.New("empty input")
	}

	// 模拟处理
	if input == "bad" {
		return "", errors.New("bad input")
	}

	return "processed: " + input, nil
}

func processWithRetry(input string, maxRetries int) (string, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		result, err := processWithFastFail(input)
		if err == nil {
			return result, nil
		}
		lastErr = err
		fmt.Printf("第 %d 次重试失败: %v\n", i+1, err)
	}

	return "", fmt.Errorf("all %d retries failed, last error: %w", maxRetries, lastErr)
}
