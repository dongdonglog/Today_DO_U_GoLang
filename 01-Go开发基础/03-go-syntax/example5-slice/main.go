package main

import "fmt"

func main() {
	// ========================================
	// 1. Slice 创建方式
	// ========================================
	fmt.Println("=== Slice 创建方式 ===")

	// 方式 1：字面量
	s1 := []int{1, 2, 3}
	fmt.Printf("字面量: %v, len=%d, cap=%d\n", s1, len(s1), cap(s1))

	// 方式 2:make 创建（推荐，可指定容量）
	s2 := make([]int, 3)    // len=3, cap=3
	s3 := make([]int, 3, 5) // len=3, cap=5
	fmt.Printf("make: %v, len=%d, cap=%d\n", s2, len(s2), cap(s2))
	fmt.Printf("make with cap: %v, len=%d, cap=%d\n", s3, len(s3), cap(s3))

	// 方式 3：从数组切割
	arr := [5]int{1, 2, 3, 4, 5}
	s4 := arr[1:4] // [2, 3, 4]
	fmt.Printf("从数组切割: %v, len=%d, cap=%d\n", s4, len(s4), cap(s4))

	// 方式 4：从 slice 切割
	s5 := s1[1:2] // [2]
	fmt.Printf("从 slice 切割: %v, len=%d, cap=%d\n", s5, len(s5), cap(s5))

	// ========================================
	// 2. 长度 vs 容量
	// ========================================
	fmt.Println("\n=== 长度 vs 容量 ===")

	s := make([]int, 0, 5) // len=0, cap=5
	fmt.Printf("初始: %v, len=%d, cap=%d\n", s, len(s), cap(s))

	s = append(s, 1)
	fmt.Printf("append 1: %v, len=%d, cap=%d\n", s, len(s), cap(s))

	s = append(s, 2, 3, 4)
	fmt.Printf("append 2,3,4: %v, len=%d, cap=%d\n", s, len(s), cap(s))

	s = append(s, 5)
	fmt.Printf("append 5: %v, len=%d, cap=%d\n", s, len(s), cap(s))

	// 容量满了，扩容
	s = append(s, 6)
	fmt.Printf("append 6 (扩容): %v, len=%d, cap=%d\n", s, len(s), cap(s))

	// ========================================
	// 3. append 扩容策略
	// ========================================
	fmt.Println("\n=== append 扩容策略 ===")

	showCapacity := func(n int) {
		s := make([]int, 0)
		for i := 0; i < n; i++ {
			s = append(s, i)
			if i == 0 || i == 1 || i == 3 || i == 7 || i == 15 || i == n-1 {
				fmt.Printf("  len=%d, cap=%d\n", len(s), cap(s))
			}
		}
	}
	showCapacity(20)

	// ========================================
	// 4. Slice 陷阱：共享底层数组
	// ========================================
	fmt.Println("\n=== Slice 陷阱：共享底层数组 ===")

	// 陷阱 1：从 slice 切割
	original := []int{1, 2, 3, 4, 5}
	slice1 := original[1:4] // [2, 3, 4]
	slice2 := original[2:5] // [3, 4, 5]

	fmt.Printf("original: %v\n", original)
	fmt.Printf("slice1: %v\n", slice1)
	fmt.Printf("slice2: %v\n", slice2)

	// 修改 original，slice1 和 slice2 也会变
	original[2] = 99
	fmt.Printf("修改 original[2]=99 后:\n")
	fmt.Printf("  original: %v\n", original)
	fmt.Printf("  slice1: %v\n", slice1)
	fmt.Printf("  slice2: %v\n", slice2)

	// 陷阱 2：append 可能覆盖
	a := make([]int, 0, 3)
	b := append(a, 1)
	c := append(a, 2)

	fmt.Printf("\nappend 陷阱:\n")
	fmt.Printf("  b: %v\n", b)
	fmt.Printf("  c: %v\n", c)
	// b 和 c 共享底层数组，c 的 append 覆盖了 b 的值

	// 正确做法：使用 copy
	fmt.Println("\n正确做法：使用 copy")
	src := []int{1, 2, 3}
	dst := make([]int, len(src))
	copy(dst, src)
	fmt.Printf("  src: %v\n", src)
	fmt.Printf("  dst: %v\n", dst)

	dst[0] = 99
	fmt.Printf("修改 dst[0]=99 后:\n")
	fmt.Printf("  src: %v (不变)\n", src)
	fmt.Printf("  dst: %v\n", dst)

	// ========================================
	// 5. Slice 底层结构
	// ========================================
	fmt.Println("\n=== Slice 底层结构 ===")

	/*
		type slice struct {
			array unsafe.Pointer // 指向底层数组
			len   int            // 长度
			cap   int            // 容量
		}

		┌─────────────────────────────────────┐
		│  Slice Header (24 bytes)            │
		│  ┌─────────────────────────────┐    │
		│  │ array: pointer (8 bytes)    │    │
		│  │ len: int (8 bytes)          │    │
		│  │ cap: int (8 bytes)          │    │
		│  └─────────────────────────────┘    │
		└─────────────────────────────────────┘
		              │
		              ▼
		┌─────────────────────────────────────┐
		│  Underlying Array                   │
		│  [0] [1] [2] [3] [4]               │
		└─────────────────────────────────────┘
	*/

	s = []int{10, 20, 30, 40, 50}
	sub := s[1:3] // [20, 30]

	fmt.Printf("s: %v, len=%d, cap=%d\n", s, len(s), cap(s))
	fmt.Printf("sub: %v, len=%d, cap=%d\n", sub, len(sub), cap(sub))
	fmt.Println("sub 和 s 共享底层数组")

	// ========================================
	// 6. 常用操作
	// ========================================
	fmt.Println("\n=== 常用操作 ===")

	data := []int{5, 2, 8, 1, 9}

	// 遍历
	fmt.Print("遍历: ")
	for _, v := range data {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// 查找
	target := 8
	for i, v := range data {
		if v == target {
			fmt.Printf("找到 %d 在索引 %d\n", target, i)
			break
		}
	}

	// 删除元素（保持顺序）
	fmt.Printf("删除前: %v\n", data)
	index := 2
	data = append(data[:index], data[index+1:]...)
	fmt.Printf("删除索引 %d 后: %v\n", index, data)

	// 清空 slice
	data = data[:0]
	fmt.Printf("清空后: %v, len=%d, cap=%d\n", data, len(data), cap(data))

	// 预分配容量（性能优化）
	fmt.Println("\n预分配容量:")
	n := 1000
	// 不预分配
	s1 = make([]int, 0)
	for i := 0; i < n; i++ {
		s1 = append(s1, i)
	}
	// 预分配
	s2 = make([]int, 0, n)
	for i := 0; i < n; i++ {
		s2 = append(s2, i)
	}
	fmt.Printf("不预分配: len=%d, cap=%d\n", len(s1), cap(s1))
	fmt.Printf("预分配: len=%d, cap=%d\n", len(s2), cap(s2))
}
