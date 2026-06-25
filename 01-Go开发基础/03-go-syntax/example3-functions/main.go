package main

import (
	"fmt"
	"os"
)

// ========================================
// 1. 多返回值
// ========================================

// 普通多返回值
func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// 命名返回值
func divideNamed(a, b float64) (result float64, err error) {
	if b == 0 {
		err = fmt.Errorf("division by zero")
		return // 裸 return，返回命名变量
	}
	result = a / b
	return
}

// ========================================
// 2. defer
// ========================================

func deferDemo() {
	fmt.Println("\n=== defer 执行顺序（LIFO）===")

	fmt.Println("开始")
	defer fmt.Println("defer 1")
	defer fmt.Println("defer 2")
	defer fmt.Println("defer 3")
	fmt.Println("结束")
	// 输出顺序：开始 -> 结束 -> defer 3 -> defer 2 -> defer 1
}

func deferFileDemo() {
	fmt.Println("\n=== defer 资源清理 ===")

	file, err := os.Open("/dev/null")
	if err != nil {
		fmt.Printf("打开文件失败: %v\n", err)
		return
	}
	defer file.Close() // 确保文件关闭

	fmt.Println("文件已打开，defer 会在函数退出时关闭")
}

// defer 陷阱：参数预计算
func deferTrap() {
	fmt.Println("\n=== defer 陷阱：参数预计算 ===")

	i := 0
	defer fmt.Printf("defer: i = %d\n", i) // i 在 defer 时就确定了，值为 0

	i = 100
	fmt.Printf("函数结束前: i = %d\n", i)
	// defer 输出: i = 0（不是 100）
}

// defer 陷阱：循环中的 defer
func deferLoopTrap() {
	fmt.Println("\n=== defer 陷阱：循环中的 defer ===")

	files := []string{"file1", "file2", "file3"}
	for _, f := range files {
		// 错误：所有 defer 在函数结束时才执行，可能导致资源耗尽
		defer fmt.Printf("关闭: %s\n", f)
	}
	fmt.Println("循环结束")
	// 输出顺序：循环结束 -> 关闭: file3 -> 关闭: file2 -> 关闭: file1
}

// 正确做法：用函数包装
func deferLoopCorrect() {
	fmt.Println("\n=== 正确做法：函数包装 ===")

	files := []string{"file1", "file2", "file3"}
	for _, f := range files {
		func(name string) {
			defer fmt.Printf("关闭: %s\n", name)
			fmt.Printf("处理: %s\n", name)
		}(f)
	}
}

// ========================================
// 3. 闭包
// ========================================

// 闭包：函数 + 引用的外部变量
func counter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

// 闭包陷阱：循环变量
func closureTrap() {
	fmt.Println("\n=== 闭包陷阱：循环变量 ===")

	fns := make([]func(), 3)

	// Go 1.22 之前：输出 3 3 3
	// Go 1.22 之后：输出 0 1 2（循环变量语义变更）
	for i := 0; i < 3; i++ {
		fns[i] = func() {
			fmt.Printf("  i = %d\n", i)
		}
	}

	for _, fn := range fns {
		fn()
	}
}

// 正确做法：参数传递
func closureCorrect() {
	fmt.Println("\n=== 正确做法：参数传递 ===")

	fns := make([]func(), 3)

	for i := 0; i < 3; i++ {
		fns[i] = func(n int) func() {
			return func() {
				fmt.Printf("  i = %d\n", n)
			}
		}(i) // 立即传参
	}

	for _, fn := range fns {
		fn()
	}
}

func main() {
	// ========================================
	// 多返回值
	// ========================================
	fmt.Println("=== 多返回值 ===")

	result, err := divide(10, 3)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("10 / 3 = %.2f\n", result)
	}

	result, err = divide(10, 0)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	// 忽略某个返回值
	result, _ = divide(10, 3)
	fmt.Printf("忽略错误: %.2f\n", result)

	// 命名返回值
	result, err = divideNamed(10, 2)
	fmt.Printf("命名返回值: %.2f, err: %v\n", result, err)

	// ========================================
	// defer
	// ========================================
	deferDemo()
	deferFileDemo()
	deferTrap()
	deferLoopTrap()
	deferLoopCorrect()

	// ========================================
	// 闭包
	// ========================================
	fmt.Println("\n=== 闭包 ===")

	c := counter()
	fmt.Printf("c() = %d\n", c()) // 1
	fmt.Printf("c() = %d\n", c()) // 2
	fmt.Printf("c() = %d\n", c()) // 3

	c2 := counter() // 新的闭包实例
	fmt.Printf("c2() = %d\n", c2()) // 1

	closureTrap()
	closureCorrect()
}
