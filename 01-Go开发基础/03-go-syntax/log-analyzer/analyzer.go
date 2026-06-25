package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// Analyzer 日志分析器
type Analyzer struct {
	counters map[string]int
	mu       sync.Mutex
}

// NewAnalyzer 创建分析器
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		counters: make(map[string]int),
	}
}

// AnalyzeLine 分析单行日志
func (a *Analyzer) AnalyzeLine(line string) {
	entry := ParseLogLine(line)
	if entry == nil || entry.Level != "ERROR" {
		return
	}

	// 提取错误类型
	errorType := extractErrorType(entry.Message)
	if errorType == "" {
		return
	}

	a.mu.Lock()
	a.counters[errorType]++
	a.mu.Unlock()
}

// AnalyzeReader 分析整个 reader
func (a *Analyzer) AnalyzeReader(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		a.AnalyzeLine(scanner.Text())
	}
	return scanner.Err()
}

// TopN 获取 Top N 错误
func (a *Analyzer) TopN(n int) []ErrorStats {
	a.mu.Lock()
	defer a.mu.Unlock()

	stats := make([]ErrorStats, 0, len(a.counters))
	for errorType, count := range a.counters {
		stats = append(stats, ErrorStats{
			ErrorType: errorType,
			Count:     count,
		})
	}

	// 按计数降序排序
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count == stats[j].Count {
			return stats[i].ErrorType < stats[j].ErrorType
		}
		return stats[i].Count > stats[j].Count
	})

	if n > len(stats) {
		n = len(stats)
	}
	return stats[:n]
}

// TotalErrors 获取总错误数
func (a *Analyzer) TotalErrors() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	total := 0
	for _, count := range a.counters {
		total += count
	}
	return total
}

// PrintReport 打印报告
func (a *Analyzer) PrintReport(n int) {
	fmt.Println("========================================")
	fmt.Println("        Error Statistics Report")
	fmt.Println("========================================")
	fmt.Printf("Total Errors: %d\n", a.TotalErrors())
	fmt.Println("----------------------------------------")
	fmt.Printf("Top %d Errors:\n", n)

	stats := a.TopN(n)
	for i, stat := range stats {
		fmt.Printf("  %d. %-30s %d\n", i+1, stat.ErrorType, stat.Count)
	}
	fmt.Println("========================================")
}

// extractErrorType 从消息中提取错误类型
func extractErrorType(message string) string {
	// 格式: "ERROR_TYPE: description"
	parts := strings.SplitN(message, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
