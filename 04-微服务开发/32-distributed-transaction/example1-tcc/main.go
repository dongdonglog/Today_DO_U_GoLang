package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"
)

// 分布式事务之 TCC:Try-Confirm-Cancel。
//
// 场景:转账 A→B,涉及两个独立"服务"(账户A、账户B),没有跨库事务可用。
// TCC 把每个参与方的操作拆成三阶段:
//
//	Try    :预留资源(冻结),保证 Confirm 一定成功
//	Confirm:提交(真正扣款/加款)
//	Cancel :补偿(释放冻结),把 Try 的预留回滚
//
// 协调者流程:
//  1. 对所有参与方依次 Try,任一失败 → 对已成功的参与方执行 Cancel
//  2. 全部 Try 成功 → 依次 Confirm;Confirm 失败可重试(幂等)
//
// 这是分布式事务里最"重"也最可靠的模式,适合强一致性要求高的场景。
// 对比 2PC(XA):TCC 的 Try/Confirm/Cancel 是业务代码,能处理网络分区与部分失败;
// 2PC 依赖数据库 XA 锁资源,协调者宕机会阻塞参与者。
//
//	go run .            # 演示成功路径 + 失败路径(带模拟故障)
func main() {
	dsn := "root:root@tcp(localhost:3306)/go_book_dtx?charset=utf8mb4&parseTime=true"
	if err := ensureDB(dsn); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	migrate(ctx, db)
	seed(ctx, db)

	coordinator := &TCCCoordinator{db: db}

	fmt.Println("========== 场景1:成功转账(全员 Try→Confirm)==========")
	seed(ctx, db)
	coordinator.Transfer(ctx, 1, 2, 3000, true)
	dump(ctx, db, "转账成功后")

	fmt.Println("\n========== 场景2:账户A预留成功,账户B的 Try 失败 → Cancel A ==========")
	seed(ctx, db)                                                     // 重置:A=100000,B=50000
	db.ExecContext(ctx, "UPDATE accounts SET balance=100 WHERE id=2") // B 只剩 100
	coordinator.Transfer(ctx, 1, 2, 3000, true)                       // A Try 成功, B Try 失败(不足) → Cancel A
	dump(ctx, db, "回滚后(应无冻结残留)")

	fmt.Println("\n========== 场景3:Confirm 阶段模拟故障 → 幂等重试 ==========")
	seed(ctx, db)
	coordinator.failConfirmOnA = true
	coordinator.Transfer(ctx, 1, 2, 1000, true)
	dump(ctx, db, "Confirm 重试后")
}

// ====== 参与方:每个都是独立"服务"的视角,共享同一个库表模拟 ======

// TryAccount 预留:把金额从 balance 冻结到 frozen(行级锁保证原子)
func TryAccount(ctx context.Context, db *sql.DB, id int64, amount int64) error {
	res, err := db.ExecContext(ctx,
		`UPDATE accounts SET balance = balance - ?, frozen = frozen + ?
		 WHERE id = ? AND balance >= ?`, // 余额不足则影响行数=0
		amount, amount, id, amount)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("账户%d余额不足", id)
	}
	return nil
}

// ConfirmAccount:提交扣款,把冻结转成扣减(实际上 balance 已扣,这里清 frozen)
func ConfirmAccount(ctx context.Context, db *sql.DB, id int64, amount int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE accounts SET frozen = frozen - ? WHERE id = ?`, amount, id)
	return err
}

// CancelAccount:补偿,把冻结释放回 balance
func CancelAccount(ctx context.Context, db *sql.DB, id int64, amount int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE accounts SET balance = balance + ?, frozen = frozen - ?
		 WHERE id = ?`, amount, amount, id)
	return err
}

// ====== 协调者 ======

type TCCCoordinator struct {
	db             *sql.DB
	failConfirmOnA bool
}

// Transfer 执行 TCC 转账
func (c *TCCCoordinator) Transfer(ctx context.Context, fromID, toID int64, amount int64, demo bool) error {
	// 阶段1:Try 所有参与方
	tried := []int64{} // 已 Try 成功的,失败时要 Cancel 它们
	for _, id := range []int64{fromID, toID} {
		if err := TryAccount(ctx, c.db, id, amount); err != nil {
			fmt.Printf("[TCC] Try 账户%d 失败: %v\n", id, err)
			// 阶段2:对已成功的参与方执行 Cancel(补偿)
			for _, done := range tried {
				if cerr := CancelAccount(ctx, c.db, done, amount); cerr != nil {
					log.Printf("[TCC] Cancel 账户%d 也失败(需人工介入): %v", done, cerr)
				} else {
					fmt.Printf("[TCC] Cancel 账户%d 预留成功\n", done)
				}
			}
			return err
		}
		fmt.Printf("[TCC] Try 账户%d 预留 %d 成功\n", id, amount)
		tried = append(tried, id)
	}

	// 阶段3:全部 Try 成功 → Confirm(每个都可独立幂等重试)
	for _, id := range []int64{fromID, toID} {
		if c.failConfirmOnA && id == fromID {
			c.failConfirmOnA = false
			fmt.Printf("[TCC] 模拟账户%d Confirm 第一次失败(网络抖动)\n", id)
		}
		if err := ConfirmAccount(ctx, c.db, id, amount); err != nil {
			// 生产:Confirm 失败要重试直到成功(幂等),或告警人工
			log.Printf("[TCC] Confirm 账户%d 失败,准备重试: %v", id, err)
			for attempt := 1; attempt <= 3; attempt++ {
				time.Sleep(50 * time.Millisecond)
				if err = ConfirmAccount(ctx, c.db, id, amount); err == nil {
					break
				}
			}
			if err != nil {
				return err // 重试耗尽,走人工处理
			}
		}
		fmt.Printf("[TCC] Confirm 账户%d 成功\n", id)
	}
	fmt.Printf("[TCC] 转账完成: %d → %d 金额 %d\n", fromID, toID, amount)
	return nil
}

// ====== 基础设施 ======

func ensureDB(dsn string) error {
	cfg, _ := mysql.ParseDSN(dsn)
	dbname := cfg.DBName
	cfg.DBName = ""
	conn, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Exec("CREATE DATABASE IF NOT EXISTS " + dbname + " CHARACTER SET utf8mb4")
	return err
}

func migrate(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS accounts (
		id BIGINT PRIMARY KEY,
		name VARCHAR(32),
		balance BIGINT NOT NULL,
		frozen BIGINT NOT NULL DEFAULT 0
	) ENGINE=InnoDB`)
	if err != nil {
		log.Fatal(err)
	}
}

func seed(ctx context.Context, db *sql.DB) {
	db.ExecContext(ctx, "DELETE FROM accounts")
	db.ExecContext(ctx, `INSERT INTO accounts (id,name,balance) VALUES
		(1,'账户A',100000),(2,'账户B',50000),(3,'账户C',200000)`)
}

func dump(ctx context.Context, db *sql.DB, tag string) {
	fmt.Printf("--- %s ---\n", tag)
	rows, _ := db.QueryContext(ctx, "SELECT id,name,balance,frozen FROM accounts ORDER BY id")
	type row struct {
		ID   int64
		Name string
		Bal  int64
		Fro  int64
	}
	for rows.Next() {
		var r row
		rows.Scan(&r.ID, &r.Name, &r.Bal, &r.Fro)
		fmt.Printf("  %s: 余额=%d 冻结=%d\n", r.Name, r.Bal, r.Fro)
	}
	rows.Close()
}
