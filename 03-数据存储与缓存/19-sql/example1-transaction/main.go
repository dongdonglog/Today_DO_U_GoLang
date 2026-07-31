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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := getenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/go_book_transaction?charset=utf8mb4&parseTime=true")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("连接失败:", err)
	}
	fmt.Println("连接成功")

	// 查看初始余额
	fmt.Println("\n=== 初始余额 ===")
	printBalances(ctx, db)

	// 测试转账
	fmt.Println("\n=== 测试转账：A 转给 B 100 元 ===")
	err = Transfer(ctx, db, "A", "B", 10000)
	if err != nil {
		log.Fatal("转账失败:", err)
	}
	fmt.Println("转账成功")

	// 查看转账后余额
	fmt.Println("\n=== 转账后余额 ===")
	printBalances(ctx, db)

	// 测试余额不足
	fmt.Println("\n=== 测试余额不足：A 转给 B 10000 元 ===")
	err = Transfer(ctx, db, "A", "B", 1000000)
	if err != nil {
		fmt.Println("转账失败（预期）:", err)
	}

	// 查看最终余额
	fmt.Println("\n=== 最终余额 ===")
	printBalances(ctx, db)
}

// Transfer 转账函数。金额使用“分”表示，避免使用 float64 处理钱。
func Transfer(ctx context.Context, db *sql.DB, from, to string, amountCents int64) error {
	if from == to {
		return fmt.Errorf("转出账户和收款账户不能相同")
	}
	if amountCents <= 0 {
		return fmt.Errorf("转账金额必须大于 0")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 扣钱
	result, err := tx.ExecContext(ctx,
		"UPDATE accounts SET balance = balance - ? WHERE user_id = ? AND balance >= ?",
		amountCents, from, amountCents)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected != 1 {
		return fmt.Errorf("转出账户不存在或余额不足")
	}

	// 加钱
	result, err = tx.ExecContext(ctx,
		"UPDATE accounts SET balance = balance + ? WHERE user_id = ?",
		amountCents, to)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected != 1 {
		return fmt.Errorf("收款账户不存在")
	}

	return tx.Commit()
}

func printBalances(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, "SELECT user_id, balance FROM accounts ORDER BY user_id")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		var balanceCents int64
		if err := rows.Scan(&userID, &balanceCents); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("用户 %s: %.2f 元\n", userID, float64(balanceCents)/100)
	}
	if err := rows.Err(); err != nil {
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
