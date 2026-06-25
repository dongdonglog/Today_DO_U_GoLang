package main

import (
	"fmt"
	"sort"
	"sync"
)

func main() {
	// ========================================
	// 1. Map 基础操作
	// ========================================
	fmt.Println("=== Map 基础操作 ===")

	// 创建方式
	// 方式 1：字面量
	m1 := map[string]int{
		"Alice": 30,
		"Bob":   25,
	}
	fmt.Printf("字面量: %v\n", m1)

	// 方式 2：make 创建（推荐）
	m2 := make(map[string]int)
	m2["Alice"] = 30
	m2["Bob"] = 25
	fmt.Printf("make: %v\n", m2)

	// 方式 3：零值（nil map，不能写入）
	var m3 map[string]int
	fmt.Printf("nil map: %v\n", m3)
	// m3["key"] = 1 // panic: assignment to entry in nil map

	// ========================================
	// 2. 访问和修改
	// ========================================
	fmt.Println("\n=== 访问和修改 ===")

	m := map[string]int{
		"Alice": 30,
		"Bob":   25,
	}

	// 读取
	age := m["Alice"]
	fmt.Printf("Alice 的年龄: %d\n", age)

	// 读取不存在的 key
	age = m["Charlie"] // 返回零值 0
	fmt.Printf("Charlie 的年龄（不存在）: %d\n", age)

	// 检查 key 是否存在
	if age, ok := m["Alice"]; ok {
		fmt.Printf("Alice 存在，年龄: %d\n", age)
	}

	if _, ok := m["Charlie"]; !ok {
		fmt.Println("Charlie 不存在")
	}

	// 修改
	m["Alice"] = 31
	fmt.Printf("修改后: %v\n", m)

	// 删除
	delete(m, "Bob")
	fmt.Printf("删除 Bob 后: %v\n", m)

	// 删除不存在的 key（不报错）
	delete(m, "NotExist")

	// ========================================
	// 3. 遍历（顺序不确定）
	// ========================================
	fmt.Println("\n=== 遍历（顺序不确定）===")

	m = map[string]int{
		"Alice":   30,
		"Bob":     25,
		"Charlie": 35,
		"David":   28,
	}

	fmt.Println("第一次遍历:")
	for k, v := range m {
		fmt.Printf("  %s: %d\n", k, v)
	}

	fmt.Println("第二次遍历（顺序可能不同）:")
	for k, v := range m {
		fmt.Printf("  %s: %d\n", k, v)
	}

	// 有序遍历
	fmt.Println("有序遍历（按 key 排序）:")
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, m[k])
	}

	// ========================================
	// 4. Map 陷阱
	// ========================================
	fmt.Println("\n=== Map 陷阱 ===")

	// 陷阱 1：并发读写 panic
	fmt.Println("陷阱 1：并发读写会 panic")
	fmt.Println("  （演示代码已注释，运行会 panic）")
	/*
		m := make(map[string]int)
		go func() {
			for {
				m["key"] = 1 // 写
			}
		}()
		go func() {
			for {
				_ = m["key"] // 读
			}
		}()
		time.Sleep(time.Second)
	*/

	// 陷阱 2：value 是结构体时，不能直接修改字段
	fmt.Println("\n陷阱 2：value 是结构体时，不能直接修改字段")

	type User struct {
		Name string
		Age  int
	}

	users := map[string]User{
		"Alice": {Name: "Alice", Age: 30},
	}

	// users["Alice"].Age = 31 // 编译错误！

	// 正确做法：取出 -> 修改 -> 放回
	u := users["Alice"]
	u.Age = 31
	users["Alice"] = u
	fmt.Printf("修改后: %v\n", users["Alice"])

	// 或者使用指针
	usersPtr := map[string]*User{
		"Alice": {Name: "Alice", Age: 30},
	}
	usersPtr["Alice"].Age = 31 // 可以直接修改
	fmt.Printf("指针方式: %v\n", usersPtr["Alice"])

	// ========================================
	// 5. sync.Map（并发安全）
	// ========================================
	fmt.Println("\n=== sync.Map（并发安全）===")

	var sm sync.Map

	// 存储
	sm.Store("Alice", 30)
	sm.Store("Bob", 25)

	// 读取
	if v, ok := sm.Load("Alice"); ok {
		fmt.Printf("Alice: %v\n", v)
	}

	// 删除
	sm.Delete("Bob")

	// 遍历
	fmt.Println("sync.Map 遍历:")
	sm.Range(func(key, value interface{}) bool {
		fmt.Printf("  %v: %v\n", key, value)
		return true
	})

	// LoadOrStore：不存在则存储，存在则返回
	actual, loaded := sm.LoadOrStore("Charlie", 35)
	fmt.Printf("LoadOrStore Charlie: value=%v, loaded=%v\n", actual, loaded)

	actual, loaded = sm.LoadOrStore("Alice", 99) // Alice 已存在
	fmt.Printf("LoadOrStore Alice: value=%v, loaded=%v\n", actual, loaded)

	// ========================================
	// 6. 实战：统计词频
	// ========================================
	fmt.Println("\n=== 实战：统计词频 ===")

	text := "hello world hello go world hello"
	words := []string{"hello", "world", "hello", "go", "world", "hello"}

	// 统计
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}

	fmt.Printf("文本: %q\n", text)
	fmt.Println("词频统计:")
	for word, count := range freq {
		fmt.Printf("  %q: %d\n", word, count)
	}

	// 按频率排序
	type WordCount struct {
		Word  string
		Count int
	}

	wcs := make([]WordCount, 0, len(freq))
	for w, c := range freq {
		wcs = append(wcs, WordCount{Word: w, Count: c})
	}

	sort.Slice(wcs, func(i, j int) bool {
		return wcs[i].Count > wcs[j].Count
	})

	fmt.Println("按频率排序:")
	for _, wc := range wcs {
		fmt.Printf("  %q: %d\n", wc.Word, wc.Count)
	}

	// ========================================
	// 7. Map 容量
	// ========================================
	fmt.Println("\n=== Map 容量 ===")

	// 预分配容量（性能优化）
	n := 1000

	// 不预分配
	m1 = make(map[string]int)
	for i := 0; i < n; i++ {
		m1[fmt.Sprintf("key%d", i)] = i
	}

	// 预分配
	m2 = make(map[string]int, n)
	for i := 0; i < n; i++ {
		m2[fmt.Sprintf("key%d", i)] = i
	}

	fmt.Printf("不预分配: len=%d\n", len(m1))
	fmt.Printf("预分配: len=%d\n", len(m2))
}
