package main

import "fmt"

func main() {
	// ========================================
	// 1. if 初始化语句
	// ========================================
	fmt.Println("=== if 初始化语句 ===")

	// 传统写法
	err := doSomething()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	// Go 推荐写法：err 只在 if 作用域内可见
	if err := doSomething(); err != nil {
		fmt.Printf("错误: %v\n", err)
	}
	// 这里 err 不可见

	// ========================================
	// 2. for 是唯一循环
	// ========================================
	fmt.Println("\n=== for 循环 ===")

	// 传统 for 循环
	fmt.Println("传统 for:")
	for i := 0; i < 3; i++ {
		fmt.Printf("  i = %d\n", i)
	}

	// for 替代 while
	fmt.Println("for 替代 while:")
	n := 0
	for n < 3 {
		fmt.Printf("  n = %d\n", n)
		n++
	}

	// for 替代无限循环
	fmt.Println("for 无限循环（break 退出）:")
	count := 0
	for {
		if count >= 3 {
			break
		}
		fmt.Printf("  count = %d\n", count)
		count++
	}

	// range 遍历切片
	fmt.Println("range 遍历切片:")
	names := []string{"Alice", "Bob", "Charlie"}
	for i, name := range names {
		fmt.Printf("  [%d] = %s\n", i, name)
	}

	// range 遍历 map
	fmt.Println("range 遍历 map:")
	ages := map[string]int{
		"Alice": 30,
		"Bob":   25,
	}
	for name, age := range ages {
		fmt.Printf("  %s: %d\n", name, age)
	}

	// range 遍历字符串
	fmt.Println("range 遍历字符串:")
	for i, ch := range "Go" {
		fmt.Printf("  [%d] = %c\n", i, ch)
	}

	// 只获取索引或值
	fmt.Println("只获取索引:")
	for i := range names {
		fmt.Printf("  index: %d\n", i)
	}

	fmt.Println("只获取值:")
	for _, name := range names {
		fmt.Printf("  name: %s\n", name)
	}

	// ========================================
	// 3. switch 不需要 break
	// ========================================
	fmt.Println("\n=== switch ===")

	// 基本 switch（自动 break）
	day := 3
	switch day {
	case 1:
		fmt.Println("Monday")
	case 2:
		fmt.Println("Tuesday")
	case 3:
		fmt.Println("Wednesday") // 不会继续执行 case 4
	case 4:
		fmt.Println("Thursday")
	default:
		fmt.Println("Other")
	}

	// 无条件 switch（替代 if-else）
	score := 85
	switch {
	case score >= 90:
		fmt.Println("A")
	case score >= 80:
		fmt.Println("B")
	case score >= 70:
		fmt.Println("C")
	default:
		fmt.Println("D")
	}

	// fallthrough（强制执行下一个 case）
	fmt.Println("fallthrough:")
	switch day {
	case 3:
		fmt.Println("  case 3")
		fallthrough // 强制执行 case 4
	case 4:
		fmt.Println("  case 4")
	}

	// 多值匹配
	fmt.Println("多值匹配:")
	weekday := "Tuesday"
	switch weekday {
	case "Monday", "Tuesday", "Wednesday", "Thursday", "Friday":
		fmt.Println("  工作日")
	case "Saturday", "Sunday":
		fmt.Println("  周末")
	}
}

func doSomething() error {
	return nil
}
