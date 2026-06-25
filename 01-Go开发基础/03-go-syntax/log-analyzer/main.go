package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// 命令行参数
	filePath := flag.String("file", "", "日志文件路径")
	topN := flag.Int("top", 10, "显示前 N 个错误")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("用法: log-analyzer -file <日志文件路径> [-top N]")
		fmt.Println("示例: log-analyzer -file testdata/sample.log -top 5")
		os.Exit(1)
	}

	// 打开文件
	file, err := os.Open(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开文件失败: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 创建分析器
	analyzer := NewAnalyzer()

	// 分析日志
	if err := analyzer.AnalyzeReader(file); err != nil {
		fmt.Fprintf(os.Stderr, "分析日志失败: %v\n", err)
		os.Exit(1)
	}

	// 打印报告
	analyzer.PrintReport(*topN)
}
