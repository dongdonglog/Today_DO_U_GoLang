package main

import (
	"encoding/json"
	"fmt"
)

// ========================================
// 1. 结构体定义
// ========================================

// 基本结构体
type User struct {
	ID    int    `json:"id"`    // 字段标签
	Name  string `json:"name"`
	Email string `json:"email"`
}

// 嵌入式结构体（组合）
type BaseModel struct {
	ID        int
	CreatedAt string
	UpdatedAt string
}

// Employee 嵌入 BaseModel
type Employee struct {
	BaseModel // 嵌入（匿名字段）
	Name      string
	Department string
}

// ========================================
// 2. 方法
// ========================================

// 值接收者：不会修改原始对象
func (u User) String() string {
	return fmt.Sprintf("User{ID: %d, Name: %s, Email: %s}", u.ID, u.Name, u.Email)
}

// 指针接收者：可以修改原始对象
func (u *User) UpdateEmail(newEmail string) {
	u.Email = newEmail
}

// 指针接收者：避免大结构体拷贝
func (u *User) LargeMethod() {
	// 如果 User 很大，值接收者会拷贝整个结构体
	// 指针接收者只拷贝 8 字节（指针大小）
	fmt.Printf("Processing user: %s\n", u.Name)
}

// ========================================
// 3. 组合实现复用
// ========================================

// LogAnalyzer 使用组合
type LogAnalyzer struct {
	counters map[string]int
}

// NewLogAnalyzer 构造函数
func NewLogAnalyzer() *LogAnalyzer {
	return &LogAnalyzer{
		counters: make(map[string]int),
	}
}

// Analyze 方法
func (a *LogAnalyzer) Analyze(logLine string) {
	a.counters[logLine]++
}

// TopN 方法
func (a *LogAnalyzer) TopN(n int) []LogEntry {
	entries := make([]LogEntry, 0, len(a.counters))
	for log, count := range a.counters {
		entries = append(entries, LogEntry{Log: log, Count: count})
	}

	// 简单排序（实际用 sort.Slice）
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Count < entries[j].Count {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n]
}

// LogEntry 结果
type LogEntry struct {
	Log   string
	Count int
}

// ========================================
// 4. 值接收者 vs 指针接收者
// ========================================

// Counter 值接收者示例
type Counter struct {
	value int
}

// 值接收者：不会修改原始对象
func (c Counter) Get() int {
	return c.value
}

// 值接收者：修改的是副本
func (c Counter) Increment() {
	c.value++ // 修改的是副本，原始对象不变
}

// CounterPointer 指针接收者示例
type CounterPointer struct {
	value int
}

// 指针接收者：会修改原始对象
func (c *CounterPointer) Increment() {
	c.value++ // 修改原始对象
}

func main() {
	// ========================================
	// 结构体创建
	// ========================================
	fmt.Println("=== 结构体创建 ===")

	// 方式 1：字段名初始化（推荐）
	u1 := User{
		ID:    1,
		Name:  "Alice",
		Email: "alice@example.com",
	}
	fmt.Printf("u1: %+v\n", u1)

	// 方式 2：按顺序初始化（不推荐，容易出错）
	u2 := User{2, "Bob", "bob@example.com"}
	fmt.Printf("u2: %+v\n", u2)

	// 方式 3：零值初始化
	u3 := User{}
	fmt.Printf("u3: %+v\n", u3)

	// 方式 4：指针
	u4 := &User{ID: 4, Name: "Charlie"}
	fmt.Printf("u4: %+v\n", u4)

	// ========================================
	// JSON 序列化（字段标签）
	// ========================================
	fmt.Println("\n=== JSON 序列化 ===")

	data, _ := json.MarshalIndent(u1, "", "  ")
	fmt.Println(string(data))

	// ========================================
	// 方法调用
	// ========================================
	fmt.Println("\n=== 方法调用 ===")

	// 值调用
	fmt.Println(u1.String())

	// 指针接收者方法
	u1.UpdateEmail("alice.new@example.com")
	fmt.Println(u1.String())

	// Go 会自动取地址
	u4.UpdateEmail("charlie@example.com")
	fmt.Println(u4.String())

	// ========================================
	// 嵌入（组合）
	// ========================================
	fmt.Println("\n=== 嵌入（组合）===")

	emp := Employee{
		BaseModel: BaseModel{
			ID:        1,
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-01-02",
		},
		Name:       "Alice",
		Department: "Engineering",
	}

	// 直接访问嵌入字段
	fmt.Printf("emp.ID: %d\n", emp.ID)
	fmt.Printf("emp.CreatedAt: %s\n", emp.CreatedAt)
	fmt.Printf("emp.Name: %s\n", emp.Name)
	fmt.Printf("emp.Department: %s\n", emp.Department)

	// 显式访问
	fmt.Printf("emp.BaseModel.ID: %d\n", emp.BaseModel.ID)

	// ========================================
	// 组合实现复用
	// ========================================
	fmt.Println("\n=== 组合实现复用 ===")

	analyzer := NewLogAnalyzer()
	analyzer.Analyze("ERROR_TIMEOUT")
	analyzer.Analyze("ERROR_TIMEOUT")
	analyzer.Analyze("ERROR_CONNECTION")
	analyzer.Analyze("ERROR_TIMEOUT")
	analyzer.Analyze("ERROR_AUTH")
	analyzer.Analyze("ERROR_CONNECTION")

	top := analyzer.TopN(2)
	fmt.Println("Top 2 errors:")
	for i, entry := range top {
		fmt.Printf("  %d. %s: %d\n", i+1, entry.Log, entry.Count)
	}

	// ========================================
	// 值接收者 vs 指针接收者
	// ========================================
	fmt.Println("\n=== 值接收者 vs 指针接收者 ===")

	c1 := Counter{value: 0}
	c1.Increment() // 值接收者，修改的是副本
	fmt.Printf("值接收者: c1.value = %d\n", c1.value) // 仍然是 0

	c2 := &CounterPointer{value: 0}
	c2.Increment() // 指针接收者，修改原始对象
	fmt.Printf("指针接收者: c2.value = %d\n", c2.value) // 变成 1
}
