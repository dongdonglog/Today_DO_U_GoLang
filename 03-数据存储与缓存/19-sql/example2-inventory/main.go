package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := getenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/go_book_inventory?charset=utf8mb4&parseTime=true")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("连接失败:", err)
	}
	fmt.Println("连接成功")

	// 查看初始库存
	fmt.Println("\n=== 初始库存 ===")
	printStock(ctx, db, "iPhone 15")

	// 测试悲观锁
	fmt.Println("\n=== 测试悲观锁（20 个并发请求）===")
	testPessimistic(ctx, db)

	// 查看库存
	fmt.Println("\n=== 悲观锁后库存 ===")
	printStock(ctx, db, "iPhone 15")

	// 重置库存
	resetStock(ctx, db, "iPhone 15", 10)

	// 测试乐观锁
	fmt.Println("\n=== 测试乐观锁（20 个并发请求）===")
	testOptimistic(ctx, db)

	// 查看库存
	fmt.Println("\n=== 乐观锁后库存 ===")
	printStock(ctx, db, "iPhone 15")
}

// DeductStockPessimistic 悲观锁扣减库存
func DeductStockPessimistic(ctx context.Context, db *sql.DB, productID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 加锁查询
	var stock int
	err = tx.QueryRowContext(ctx,
		"SELECT stock FROM products WHERE id = ? FOR UPDATE",
		productID).Scan(&stock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("商品不存在: %s", productID)
		}
		return err
	}

	if stock <= 0 {
		return fmt.Errorf("库存不足")
	}

	// 扣减库存
	result, err := tx.ExecContext(ctx,
		"UPDATE products SET stock = stock - 1 WHERE id = ?", productID)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected != 1 {
		return fmt.Errorf("商品不存在: %s", productID)
	}

	return tx.Commit()
}

// DeductStockOptimistic 乐观锁扣减库存
func DeductStockOptimistic(ctx context.Context, db *sql.DB, productID string) error {
	for retry := 0; retry < 3; retry++ {
		// 查询库存和版本号
		var stock, version int
		err := db.QueryRowContext(ctx,
			"SELECT stock, version FROM products WHERE id = ?",
			productID).Scan(&stock, &version)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("商品不存在: %s", productID)
			}
			return err
		}

		if stock <= 0 {
			return fmt.Errorf("库存不足")
		}

		// 带版本号更新
		result, err := db.ExecContext(ctx,
			"UPDATE products SET stock = stock - 1, version = version + 1 WHERE id = ? AND version = ?",
			productID, version)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected > 0 {
			return nil
		}

		// 版本号冲突，重试
		if retry < 2 {
			timer := time.NewTimer(time.Duration(retry+1) * 20 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		fmt.Printf("  版本号冲突，重试第 %d 次\n", retry+1)
	}
	return fmt.Errorf("更新失败，请重试")
}

func testPessimistic(ctx context.Context, db *sql.DB) {
	var wg sync.WaitGroup
	success := 0
	fail := 0
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := DeductStockPessimistic(ctx, db, "1")
			mu.Lock()
			if err != nil {
				fail++
			} else {
				success++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("成功: %d, 失败: %d, 耗时: %v\n", success, fail, time.Since(start))
}

func testOptimistic(ctx context.Context, db *sql.DB) {
	var wg sync.WaitGroup
	success := 0
	fail := 0
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := DeductStockOptimistic(ctx, db, "1")
			mu.Lock()
			if err != nil {
				fail++
			} else {
				success++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("成功: %d, 失败: %d, 耗时: %v\n", success, fail, time.Since(start))
}

func printStock(ctx context.Context, db *sql.DB, name string) {
	var stock, version int
	if err := db.QueryRowContext(ctx,
		"SELECT stock, version FROM products WHERE name = ?", name).
		Scan(&stock, &version); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("商品: %s, 库存: %d, 版本: %d\n", name, stock, version)
}

func resetStock(ctx context.Context, db *sql.DB, name string, stock int) {
	if _, err := db.ExecContext(ctx,
		"UPDATE products SET stock = ?, version = 0 WHERE name = ?",
		stock, name); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
