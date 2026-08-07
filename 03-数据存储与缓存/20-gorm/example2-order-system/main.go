package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ensureDatabase 连接到 MySQL 服务端并创建演示数据库
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
	_, err = sqldb.Exec("CREATE DATABASE IF NOT EXISTS " + strings.ReplaceAll(dbname, "`", "``") + " CHARACTER SET utf8mb4")
	return err
}

// User 用户模型
type User struct {
	ID        uint            `gorm:"primaryKey"`
	Username  string          `gorm:"size:50;not null"`
	Email     string          `gorm:"size:100;not null"`
	Balance   decimal.Decimal `gorm:"type:decimal(10,2);default:0"`
	Version   int             `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Orders    []Order        `gorm:"foreignKey:UserID"`
}

// Order 订单模型
type Order struct {
	ID        uint            `gorm:"primaryKey"`
	OrderNo   string          `gorm:"size:64;not null;uniqueIndex"`
	UserID    uint            `gorm:"not null;index"`
	Status    int8            `gorm:"default:0;comment:0:待支付 1:已支付 2:已完成"`
	Total     decimal.Decimal `gorm:"type:decimal(10,2);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	User      User        `gorm:"foreignKey:UserID"`
	Items     []OrderItem `gorm:"foreignKey:OrderID"`
}

// OrderItem 订单明细
type OrderItem struct {
	ID        uint            `gorm:"primaryKey"`
	OrderID   uint            `gorm:"not null;index"`
	ProductID uint            `gorm:"not null"`
	Quantity  int             `gorm:"not null"`
	Price     decimal.Decimal `gorm:"type:decimal(10,2);not null"`
}

// UserRepository 用户仓储接口
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uint) (*User, error)
	List(ctx context.Context, req ListUsersRequest) ([]User, int64, error)
	DeductBalance(ctx context.Context, userID uint, amount decimal.Decimal) error
}

type ListUsersRequest struct {
	Page int
	Size int
}

// userRepository 基于泛型 API 实现：repo 内持有 *gorm.DB，每个方法用 gorm.G[User](db) 构造类型化查询
type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *User) error {
	// Create 接收 *T，ID 自动回填
	return gorm.G[User](r.db).Create(ctx, user)
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*User, error) {
	// Preload 用闭包在 PreloadBuilder 上加条件；nil 表示默认预加载全部
	u, err := gorm.G[User](r.db).Preload("Orders", nil).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) List(ctx context.Context, req ListUsersRequest) ([]User, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Size > 100 {
		req.Size = 100
	}

	// Count 接收列名；查询条件需复用，所以拆成两次构造
	users := gorm.G[User](r.db)
	total, err := users.Count(ctx, "id")
	if err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.Size
	list, err := gorm.G[User](r.db).
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(req.Size).
		Find(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// DeductBalance 扣减余额（乐观锁）
func (r *userRepository) DeductBalance(ctx context.Context, userID uint, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("扣减金额必须大于 0")
	}

	for i := 0; i < 3; i++ {
		// 读出当前版本号
		user, err := gorm.G[User](r.db).Where("id = ?", userID).First(ctx)
		if err != nil {
			return err
		}

		if user.Balance.Cmp(amount) < 0 {
			return errors.New("余额不足")
		}

		// 带表达式的多列更新（balance = balance - ?、version = version + 1）用 Exec：
		// 泛型 Updates 只接收 T 结构体，无法直接传 gorm.Expr；Exec 仍走 GORM 连接与占位符转义
		res := r.db.WithContext(ctx).Exec(
			"UPDATE users SET balance = balance - ?, version = version + 1 WHERE id = ? AND version = ? AND deleted_at IS NULL",
			amount, userID, user.Version,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			return nil // 更新成功
		}

		if i < 2 {
			timer := time.NewTimer(time.Duration(i+1) * 50 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return errors.New("更新失败，请重试")
}

func main() {
	dsn := getenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/go_book_gorm2?charset=utf8mb4&parseTime=true")

	// 演示程序自建数据库，保证可独立运行
	if err := ensureDatabase(dsn); err != nil {
		log.Fatal("建库失败:", err)
	}

	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	if err := db.AutoMigrate(&User{}, &Order{}, &OrderItem{}); err != nil {
		log.Fatal("迁移失败:", err)
	}

	ctx := context.Background()
	alice, bob := seed(ctx, db)

	repo := NewUserRepository(db)

	// 1. 查询用户（带订单）
	fmt.Println("\n=== 1. 查询用户（带订单）===")
	user, err := repo.GetByID(ctx, alice.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("用户: %s, 订单数: %d\n", user.Username, len(user.Orders))
	for _, order := range user.Orders {
		fmt.Printf("  - 订单 %s: %s 元\n", order.OrderNo, order.Total.StringFixed(2))
	}

	// 2. 分页查询
	fmt.Println("\n=== 2. 分页查询 ===")
	users, total, err := repo.List(ctx, ListUsersRequest{Page: 1, Size: 2})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("总数: %d, 当前页: %d\n", total, len(users))
	for _, u := range users {
		fmt.Printf("  - %s (余额 %s)\n", u.Username, u.Balance.StringFixed(2))
	}

	// 3. 乐观锁扣减余额
	fmt.Println("\n=== 3. 乐观锁扣减余额 ===")
	before, _ := repo.GetByID(ctx, alice.ID)
	if err := repo.DeductBalance(ctx, alice.ID, decimal.NewFromFloat(100.00)); err != nil {
		log.Fatal(err)
	}
	after, _ := repo.GetByID(ctx, alice.ID)
	fmt.Printf("扣减前余额: %s, 扣减后余额: %s (version %d -> %d)\n",
		before.Balance.StringFixed(2), after.Balance.StringFixed(2), before.Version, after.Version)

	// 4. 事务示例（事务里可以直接用泛型 API：gorm.G[User](tx)）
	fmt.Println("\n=== 4. 事务示例 ===")
	err = db.Transaction(func(tx *gorm.DB) error {
		if _, err := gorm.G[User](tx).Where("id = ?", alice.ID).Update(ctx, "balance", gorm.Expr("balance - ?", 50)); err != nil {
			return err
		}
		if _, err := gorm.G[User](tx).Where("id = ?", bob.ID).Update(ctx, "balance", gorm.Expr("balance + ?", 50)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal("事务失败:", err)
	}
	fmt.Println("转账成功")
}

// seed 准备演示数据：两个用户 + 一个订单，返回创建后的用户（含自增 ID）
func seed(ctx context.Context, db *gorm.DB) (*User, *User) {
	// 物理清理上次演示数据。泛型 API 的 Scopes(Unscoped) 不保证级联到删除，
	// 这里直接用底层链式 API 的 Unscoped() 最稳妥
	db.WithContext(ctx).Unscoped().Where("order_no LIKE ?", "NO-GORM-DEMO-%").Delete(&Order{})
	db.WithContext(ctx).Unscoped().Where("username LIKE ?", "gorm_order_%").Delete(&User{})

	alice := &User{Username: "gorm_order_alice", Email: "alice@example.com", Balance: decimal.NewFromFloat(1000.00)}
	bob := &User{Username: "gorm_order_bob", Email: "bob@example.com", Balance: decimal.NewFromFloat(500.00)}
	if err := gorm.G[User](db).Create(ctx, alice); err != nil {
		log.Fatal("seed user:", err)
	}
	if err := gorm.G[User](db).Create(ctx, bob); err != nil {
		log.Fatal("seed user:", err)
	}
	order := &Order{
		OrderNo: fmt.Sprintf("NO-GORM-DEMO-%d", time.Now().Unix()),
		UserID:  alice.ID,
		Status:  1,
		Total:   decimal.NewFromFloat(199.00),
	}
	if err := gorm.G[Order](db).Create(ctx, order); err != nil {
		log.Fatal("seed order:", err)
	}
	return alice, bob
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
