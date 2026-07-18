package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := getenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/go_book_sql?charset=utf8mb4&parseTime=true")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("连接失败:", err)
	}
	fmt.Println("连接成功")

	// 1. 慢查询演示
	fmt.Println("\n=== 1. 慢查询演示 ===")
	slowQuery(ctx, db)

	// 2. EXPLAIN 分析
	fmt.Println("\n=== 2. EXPLAIN 分析 ===")
	explainQuery(ctx, db)

	// 3. COUNT 对比
	fmt.Println("\n=== 3. COUNT 对比 ===")
	countCompare(ctx, db)
}

func slowQuery(ctx context.Context, db *sql.DB) {
	query := `SELECT * FROM orders WHERE status = 0 AND created_at > '2025-01-01' ORDER BY created_at DESC LIMIT 20 OFFSET 1000`

	now := time.Now()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Fatal("查询失败:", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		count++
	}
	fmt.Printf("查询返回 %d 行, 耗时: %v\n", count, time.Since(now))
}

func explainQuery(ctx context.Context, db *sql.DB) {
	query := `EXPLAIN SELECT * FROM orders WHERE status = 0 AND created_at > '2025-01-01' ORDER BY created_at DESC LIMIT 20 OFFSET 1000`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Fatal("EXPLAIN 失败:", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	vals := make([]any, len(cols))
	valPtrs := make([]any, len(cols))
	for i := range vals {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		rows.Scan(valPtrs...)
		fmt.Println("EXPLAIN 结果:")
		for i, col := range cols {
			var valStr string
			switch v := vals[i].(type) {
			case []byte:
				valStr = string(v)
			case nil:
				valStr = "NULL"
			default:
				valStr = fmt.Sprintf("%v", v)
			}
			fmt.Printf("  %-15s: %s\n", col, valStr)
		}
	}
}

func countCompare(ctx context.Context, db *sql.DB) {
	var total int

	now := time.Now()
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders`).Scan(&total)
	fmt.Printf("COUNT(*) 全表: %d, 耗时: %v\n", total, time.Since(now))

	now = time.Now()
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE status = 0`).Scan(&total)
	fmt.Printf("COUNT(*) WHERE status=0: %d, 耗时: %v\n", total, time.Since(now))
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
