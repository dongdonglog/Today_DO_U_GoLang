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

type Order struct {
	ID          int64
	OrderNo     string
	UserID      int64
	Status      int
	TotalAmount float64
	CreatedAt   time.Time
}

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

	// 分页优化对比
	fmt.Println("\n=== 分页优化对比 ===")
	paginationCompare(ctx, db)
}

func paginationCompare(ctx context.Context, db *sql.DB) {
	offsets := []int{0, 1000, 10000, 50000, 100000}

	fmt.Println("--- 普通分页 ---")
	for _, offset := range offsets {
		now := time.Now()
		_, err := normalPagination(ctx, db, offset)
		if err != nil {
			log.Printf("分页查询失败: %v", err)
			continue
		}
		fmt.Printf("LIMIT 20 OFFSET %-7d 耗时: %v\n", offset, time.Since(now))
	}

	fmt.Println("\n--- 延迟关联 ---")
	for _, offset := range offsets {
		now := time.Now()
		_, err := deferredJoinPagination(ctx, db, offset)
		if err != nil {
			log.Printf("延迟关联查询失败: %v", err)
			continue
		}
		fmt.Printf("延迟关联 OFFSET %-7d 耗时: %v\n", offset, time.Since(now))
	}
}

func normalPagination(ctx context.Context, db *sql.DB, offset int) ([]Order, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, order_no, user_id, status, total_amount, created_at
		FROM orders
		WHERE status = 0
		ORDER BY created_at DESC
		LIMIT 20 OFFSET ?`, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		rows.Scan(&o.ID, &o.OrderNo, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt)
		orders = append(orders, o)
	}
	return orders, nil
}

func deferredJoinPagination(ctx context.Context, db *sql.DB, offset int) ([]Order, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT o.id, o.order_no, o.user_id, o.status, o.total_amount, o.created_at
		FROM orders o
		INNER JOIN (
			SELECT id FROM orders
			WHERE status = 0
			ORDER BY created_at DESC
			LIMIT 20 OFFSET ?
		) tmp ON o.id = tmp.id
		ORDER BY o.created_at DESC`, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		rows.Scan(&o.ID, &o.OrderNo, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt)
		orders = append(orders, o)
	}
	return orders, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
