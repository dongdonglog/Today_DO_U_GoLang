package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Outbox 模式:解决"业务写库成功、但发消息失败/漏发"的一致性问题。
//
// 反模式(先落库再发消息):
//
//	db.Commit() 成功 → kafka.Publish() 失败 → 库里状态变了,事件永远丢了
//
// Outbox 的做法:业务数据和事件在【同一个本地事务】里写入,
// 再由后台 relay 把 outbox 表里的事件投递出去。
//
//   - 业务与事件要么都写成功,要么都回滚 —— 不存在"状态变了没事件"
//
//   - relay 投递成功后才标记 sent;崩溃重启会重投 => 下游仍需幂等(at-least-once)
//
//     go run . -role=init      # 建表
//     go run . -role=business  # 模拟下单:orders + outbox 同事务写入
//     go run . -role=relay     # 启动中继,把未投递事件"发送"(演示为打印)
type orderRow struct {
	OrderNo string
	UserID  int64
	Amount  int64
}

func main() {
	role := "relay"
	for _, a := range os.Args[1:] {
		if len(a) > 6 && a[:6] == "-role=" {
			role = a[6:]
		}
	}

	dsn := "root:root@tcp(localhost:3306)/go_book_outbox?charset=utf8mb4&parseTime=true"
	// init 需要在目标库不存在时也能连上:先连服务端建库
	if role == "init" {
		if err := ensureDatabase(dsn); err != nil {
			log.Fatal(err)
		}
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	switch role {
	case "init":
		migrate(ctx, db)
	case "business":
		business(ctx, db)
	default:
		relay(ctx, db)
	}
}

func migrate(ctx context.Context, db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS orders (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			order_no VARCHAR(64) NOT NULL UNIQUE,
			user_id BIGINT NOT NULL,
			amount BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'created',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB`,
		`CREATE TABLE IF NOT EXISTS outbox (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			agg_type VARCHAR(32) NOT NULL,
			agg_id VARCHAR(64) NOT NULL,
			event_type VARCHAR(64) NOT NULL,
			payload JSON NOT NULL,
			sent TINYINT NOT NULL DEFAULT 0,
			retry_count INT NOT NULL DEFAULT 0,
			next_retry_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			KEY idx_unsent (sent, next_retry_at)
		) ENGINE=InnoDB`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			log.Fatalf("迁移失败: %v\n%s", err, s)
		}
	}
	fmt.Println("表结构就绪")
}

// business 模拟下单业务:orders 与 outbox 在同一事务提交
func business(ctx context.Context, db *sql.DB) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback() // 出错回滚;成功后显式 Commit

	orderNo := fmt.Sprintf("NO-OB-%d", time.Now().UnixNano()%100000)
	amount := int64(19900)

	res, err := tx.ExecContext(ctx,
		`INSERT INTO orders (order_no, user_id, amount) VALUES (?, ?, ?)`,
		orderNo, int64(42), amount)
	if err != nil {
		log.Fatal(err)
	}
	orderID, _ := res.LastInsertId()

	// 关键:事件和业务数据同一个事务。这里"发消息"只是往自己的表里插一行,
	// 它和 UPDATE orders 的成败绑定在一起 —— 这就是可靠性的来源
	event, _ := json.Marshal(map[string]any{
		"order_no": orderNo, "user_id": 42, "amount": amount, "action": "created",
	})
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO outbox (agg_type, agg_id, event_type, payload)
		 VALUES ('order', ?, 'order.created', ?)`, fmt.Sprint(orderID), string(event)); err != nil {
		log.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("下单成功 %s(order id=%d),事件已随事务写入 outbox\n", orderNo, orderID)
}

// ensureDatabase 解析 DSN 并创建目标数据库(init 自举用)
func ensureDatabase(dsn string) error {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	dbname := cfg.DBName
	cfg.DBName = ""
	sqldb, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer sqldb.Close()
	_, err = sqldb.ExecContext(context.Background(), "CREATE DATABASE IF NOT EXISTS "+dbname+" CHARACTER SET utf8mb4")
	return err
}

// relay 扫描未投递事件,"发送"(打印模拟),成功后标记 sent=1
func relay(ctx context.Context, db *sql.DB) {
	fmt.Println("relay 启动,每秒扫描一次 outbox,Ctrl+C 退出")
	for {
		rows, err := db.QueryContext(ctx, `
			SELECT id, agg_id, event_type, payload FROM outbox
			WHERE sent = 0 AND next_retry_at <= NOW()
			ORDER BY id LIMIT 10`)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}

		type item struct {
			id        int64
			aggID     string
			eventType string
			payload   []byte
		}
		var batch []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.aggID, &it.eventType, &it.payload); err == nil {
				batch = append(batch, it)
			}
		}
		rows.Close()

		for _, it := range batch {
			// 真实实现:kafka ProduceSync;失败则 retry_count+1、next_retry_at 退避
			fmt.Printf("[relay] 投递 #%d %s -> %s\n",
				it.id, it.eventType, it.payload)

			if _, err := db.ExecContext(ctx,
				`UPDATE outbox SET sent = 1 WHERE id = ?`, it.id); err != nil {
				log.Println("标记失败(下轮重投):", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
