package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
)

// 分布式事务之 SAGA:长事务拆成有序的本地事务步骤,失败时逐级反向补偿。
//
// 与 TCC 的区别:
//
//	TCC 在每个参与方做 Try(预留)/Confirm/Cancel,适合强一致;
//	SAGA 没有"预留"阶段,直接执行正操作,失败后用【补偿操作】回滚已经提交的步骤,
//	适合最终一致、可补偿的业务(下单/扣库存)。
//
// 编排式(本章):一个 SagaCoordinator 依次调用步骤,失败后反向调用补偿。
// 对比 choreography(编舞式):步骤间通过消息互相触发,无中心协调者,难跟踪。
//
// 场景:下单 = ① 创建订单 → ② 扣余额 → ③ 减库存
//
//	go run .               # 演示:③库存不足 → 反向补偿②扣余额、①订单置失败
//	go run . -ok           # 演示全成功
func main() {
	ok := false
	if len(os.Args) > 1 && os.Args[1] == "-ok" {
		ok = true
	}

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
	migrateSaga(ctx, db)
	seedSaga(ctx, db)

	saga := &SagaCoordinator{db: db, failInventory: !ok}

	fmt.Println("========== SAGA 下单开始 ==========")
	if err := saga.PlaceOrder(ctx, "NO-SAGA-001", 42, 2, 2500); err != nil {
		fmt.Printf("下单失败(已补偿回滚): %v\n", err)
	} else {
		fmt.Println("下单成功")
	}
	dumpSaga(ctx, db)
}

// ====== 步骤:每个都是独立"服务"的一个本地事务 ======

// StepCreateOrder ①创建订单(状态 created)
func StepCreateOrder(ctx context.Context, db *sql.DB, orderNo string, userID int64, amount int64) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO saga_orders (order_no,user_id,amount,status) VALUES (?,?,?,'created')`,
		orderNo, userID, amount)
	return err
}

// CompensateOrder ①补偿:把订单标记为 failed(不物理删除,保留审计)
func CompensateOrder(ctx context.Context, db *sql.DB, orderNo string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE saga_orders SET status='failed' WHERE order_no=?`, orderNo)
	return err
}

// StepDeductBalance ②扣余额
func StepDeductBalance(ctx context.Context, db *sql.DB, userID, amount int64) error {
	res, err := db.ExecContext(ctx,
		`UPDATE saga_accounts SET balance=balance-? WHERE id=? AND balance>=?`,
		amount, userID, amount)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("余额不足")
	}
	return nil
}

// CompensateBalance ②补偿:加回余额
func CompensateBalance(ctx context.Context, db *sql.DB, userID, amount int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE saga_accounts SET balance=balance+? WHERE id=?`, amount, userID)
	return err
}

// StepDeductInventory ③减库存;fail=true 时模拟"库存服务故障/不足"
func StepDeductInventory(ctx context.Context, db *sql.DB, skuID int64, qty int64, fail bool) error {
	if fail {
		return fmt.Errorf("库存服务异常:SKU%d 扣减失败(模拟)", skuID)
	}
	res, err := db.ExecContext(ctx,
		`UPDATE saga_inventory SET stock=stock-? WHERE sku_id=? AND stock>=?`,
		qty, skuID, qty)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("库存不足")
	}
	return nil
}

// ====== 编排协调者 ======

type SagaCoordinator struct {
	db            *sql.DB
	failInventory bool
}

type sagaStep struct {
	name       string
	forward    func(ctx context.Context) error
	compensate func(ctx context.Context) error
}

// PlaceOrder 按序执行步骤,任一失败则反向补偿已成功的步骤
func (s *SagaCoordinator) PlaceOrder(ctx context.Context, orderNo string, userID, qty, amount int64) error {
	steps := []sagaStep{
		{
			name:       "①创建订单",
			forward:    func(c context.Context) error { return StepCreateOrder(c, s.db, orderNo, userID, amount) },
			compensate: func(c context.Context) error { return CompensateOrder(c, s.db, orderNo) },
		},
		{
			name:       "②扣余额",
			forward:    func(c context.Context) error { return StepDeductBalance(c, s.db, userID, amount) },
			compensate: func(c context.Context) error { return CompensateBalance(c, s.db, userID, amount) },
		},
		{
			name:       "③减库存",
			forward:    func(c context.Context) error { return StepDeductInventory(c, s.db, 1001, qty, s.failInventory) },
			compensate: nil, // 最后一个步骤失败不需要补偿自己
		},
	}

	done := []int{} // 已成功步骤的下标
	for i, st := range steps {
		start := time.Now()
		if err := st.forward(ctx); err != nil {
			fmt.Printf("[SAGA] %s 失败(耗时%v): %v\n", st.name, time.Since(start).Round(time.Millisecond), err)
			// 反向补偿:从最近成功的一步往前
			for j := len(done) - 1; j >= 0; j-- {
				comp := steps[done[j]].compensate
				if comp == nil {
					continue
				}
				if cerr := comp(ctx); cerr != nil {
					// 补偿失败是最坏情况:需人工介入(补偿幂等+重试)
					log.Printf("[SAGA] 补偿%s 失败,需人工处理: %v", steps[done[j]].name, cerr)
				} else {
					fmt.Printf("[SAGA] 补偿%s 成功\n", steps[done[j]].name)
				}
			}
			return err
		}
		fmt.Printf("[SAGA] %s 成功\n", st.name)
		done = append(done, i)
	}
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

func migrateSaga(ctx context.Context, db *sql.DB) {
	for _, s := range []string{
		`CREATE TABLE IF NOT EXISTS saga_orders (
			order_no VARCHAR(64) PRIMARY KEY, user_id BIGINT, amount BIGINT, status VARCHAR(16)) ENGINE=InnoDB`,
		`CREATE TABLE IF NOT EXISTS saga_accounts (
			id BIGINT PRIMARY KEY, name VARCHAR(32), balance BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE IF NOT EXISTS saga_inventory (
			sku_id BIGINT PRIMARY KEY, stock BIGINT) ENGINE=InnoDB`,
	} {
		if _, err := db.ExecContext(ctx, s); err != nil {
			log.Fatal(err)
		}
	}
}

func seedSaga(ctx context.Context, db *sql.DB) {
	db.ExecContext(ctx, "DELETE FROM saga_orders")
	db.ExecContext(ctx, "DELETE FROM saga_accounts")
	db.ExecContext(ctx, "DELETE FROM saga_inventory")
	db.ExecContext(ctx, "INSERT INTO saga_accounts (id,name,balance) VALUES (42,'用户42',50000)")
	db.ExecContext(ctx, "INSERT INTO saga_inventory (sku_id,stock) VALUES (1001,10)")
}

func dumpSaga(ctx context.Context, db *sql.DB) {
	fmt.Println("--- SAGA 最终状态 ---")
	rows, _ := db.QueryContext(ctx, "SELECT order_no,status FROM saga_orders")
	fmt.Print("  订单: ")
	for rows.Next() {
		var no, st string
		rows.Scan(&no, &st)
		fmt.Printf("%s=%s ", no, st)
	}
	rows.Close()
	fmt.Println()
	var bal, stock int64
	db.QueryRowContext(ctx, "SELECT balance FROM saga_accounts WHERE id=42").Scan(&bal)
	db.QueryRowContext(ctx, "SELECT stock FROM saga_inventory WHERE sku_id=1001").Scan(&stock)
	fmt.Printf("  余额=%d 库存=%d\n", bal, stock)
}
