package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

// ========================================
// 1. panic 基础
// ========================================

func panicBasic() {
	fmt.Println("=== panic 基础 ===")

	// panic 会立即停止当前函数的执行
	// 开始执行 defer 调用链
	// 如果 panic 没有被 recover，程序会崩溃

	defer func() {
		fmt.Println("defer 1")
	}()

	defer func() {
		fmt.Println("defer 2 - 准备 panic")
		panic("something terrible happened")
		fmt.Println("这行不会执行")
	}()

	defer func() {
		fmt.Println("defer 3")
	}()

	fmt.Println("开始执行")
}

// ========================================
// 2. recover 捕获 panic
// ========================================

func recoverDemo() {
	fmt.Println("\n=== recover 捕获 panic ===")

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("捕获到 panic: %v\n", r)
			fmt.Printf("堆栈信息:\n%s\n", debug.Stack())
		}
	}()

	panic("something went wrong")

	fmt.Println("这行不会执行")
}

// ========================================
// 3. recover 只在 defer 中有效
// ========================================

func recoverInDefer() {
	fmt.Println("\n=== recover 只在 defer 中有效 ===")

	// 正确：在 defer 中调用 recover
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("正确捕获: %v\n", r)
		}
	}()

	panic("test")
}

func recoverNotInDefer() {
	fmt.Println("\n=== recover 不在 defer 中（错误）===")

	// 错误：recover 不在 defer 中，不会捕获 panic
	if r := recover(); r != nil {
		fmt.Printf("这行不会执行: %v\n", r)
	}

	// 这会导致程序崩溃
	// panic("test") // 注释掉避免崩溃
}

// ========================================
// 4. panic 传播
// ========================================

func panicPropagation() {
	fmt.Println("\n=== panic 传播 ===")

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("在最外层捕获: %v\n", r)
		}
	}()

	funcA()
}

func funcA() {
	defer func() {
		fmt.Println("funcA defer")
	}()
	funcB()
}

func funcB() {
	defer func() {
		fmt.Println("funcB defer")
	}()
	funcC()
}

func funcC() {
	defer func() {
		fmt.Println("funcC defer")
	}()
	panic("panic from funcC")
}

// ========================================
// 5. 什么时候用 panic
// ========================================

// 场景 1：程序启动时的必要初始化
func initRequired() {
	fmt.Println("\n=== 场景 1：必要初始化 ===")

	// 如果配置文件加载失败，程序无法运行
	// 这种情况可以用 panic
	config := loadConfig()
	fmt.Printf("Config loaded: %v\n", config)
}

func loadConfig() map[string]string {
	// 模拟加载配置
	config := map[string]string{
		"port": "8080",
		"host": "localhost",
	}

	// 如果加载失败
	// panic("failed to load config")

	return config
}

// 场景 2：不应该发生的情况
func unreachableCode() {
	fmt.Println("\n=== 场景 2：不应该发生的情况 ===")

	result := divide(10, 2)
	fmt.Printf("10 / 2 = %d\n", result)

	// 这种情况不应该发生
	// divide 函数内部已经检查了除数为 0 的情况
	// 如果还是出现了，说明有 bug
}

func divide(a, b int) int {
	if b == 0 {
		// 除数为 0 是编程错误，应该 panic
		panic("division by zero")
	}
	return a / b
}

// 场景 3：临时使用（不推荐）
func temporaryPanic() {
	fmt.Println("\n=== 场景 3：临时使用（不推荐）===")

	// 有时候在开发阶段，用 panic 快速暴露问题
	// 但在生产环境中应该用 error 处理

	// TODO: 实现这个功能
	// panic("not implemented")
}

// ========================================
// 6. 什么时候不用 panic
// ========================================

// 错误场景 1：可预期的错误
func expectedError() {
	fmt.Println("\n=== 错误场景：可预期的错误 ===")

	// 错误：用 panic 处理文件不存在
	// file, err := os.Open("file.txt")
	// if err != nil {
	//     panic(err) // 错误！文件不存在是可预期的
	// }

	// 正确：用 error 处理
	file, err := os.Open("nonexistent.txt")
	if err != nil {
		fmt.Printf("文件不存在: %v\n", err)
		return
	}
	defer file.Close()
}

// 错误场景 2：API 错误
func apiError() {
	fmt.Println("\n=== 错误场景：API 错误 ===")

	// 错误：用 panic 处理参数验证
	// func createUser(name string) {
	//     if name == "" {
	//         panic("name is required") // 错误！
	//     }
	// }

	// 正确：用 error 处理
	err := createUser("")
	if err != nil {
		fmt.Printf("创建用户失败: %v\n", err)
	}
}

func createUser(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// ========================================
// 7. panic/recover 最佳实践
// ========================================

// 在 HTTP 服务器中使用 recover
func httpServerRecover() {
	fmt.Println("\n=== HTTP 服务器中的 recover ===")

	// 模拟 HTTP 处理器
	handler := func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("捕获 panic: %v\n", r)
				fmt.Printf("堆栈:\n%s\n", debug.Stack())
				// 返回 500 错误
			}
		}()

		// 模拟处理器中的 panic
		panic("handler crashed")
	}

	handler()
}

// 在 goroutine 中使用 recover
func goroutineRecover() {
	fmt.Println("\n=== goroutine 中的 recover ===")

	// 每个 goroutine 都需要自己的 recover
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("goroutine 捕获 panic: %v\n", r)
			}
		}()

		panic("goroutine crashed")
	}()

	// 等待 goroutine 执行
	// time.Sleep(100 * time.Millisecond)
}

func main() {
	// panicBasic() // 会崩溃
	recoverDemo()
	recoverInDefer()
	panicPropagation()

	initRequired()
	unreachableCode()
	temporaryPanic()

	expectedError()
	apiError()

	httpServerRecover()
	goroutineRecover()

	fmt.Println("\n=== 总结 ===")
	fmt.Println("panic 的使用场景：")
	fmt.Println("1. 程序启动时的必要初始化失败")
	fmt.Println("2. 不应该发生的情况（编程错误）")
	fmt.Println("3. 临时使用（开发阶段）")
	fmt.Println("")
	fmt.Println("不要用 panic 处理：")
	fmt.Println("1. 可预期的错误（文件不存在、参数错误等）")
	fmt.Println("2. API 错误（应该返回 error）")
	fmt.Println("3. 业务逻辑错误（应该用 error 处理）")
}
