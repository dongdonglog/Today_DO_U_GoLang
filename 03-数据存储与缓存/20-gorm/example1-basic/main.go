package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// User 用户模型
type User struct {
	ID        uint            `gorm:"primaryKey"`
	Username  string          `gorm:"size:50;not null"`
	Email     string          `gorm:"size:100;not null"`
	Password  string          `gorm:"column:password_hash;size:255;not null"`
	Phone     *string         `gorm:"size:20"`
	Status    int8            `gorm:"default:1;comment:0:禁用 1:启用"`
	Balance   decimal.Decimal `gorm:"type:decimal(10,2);default:0"`
	Version   int             `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}

func main() {
	dsn := getenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/go_book_gorm?charset=utf8mb4&parseTime=true")

	// 初始化 GORM
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatal("迁移失败:", err)
	}

	ctx := context.Background()
	// users 是泛型 Repository：编译期绑定模型，CRUD 方法都接收 ctx，返回值直接是类型化对象
	users := gorm.G[User](db)

	// 清理上一次运行的数据（物理删除，避免唯一键冲突）
	unscoped := func(stmt *gorm.Statement) { stmt.Unscoped = true }
	gorm.G[User](db).Scopes(unscoped).Where("username LIKE ?", "gorm_demo_%").Delete(ctx)

	// 1. 创建
	fmt.Println("\n=== 1. 创建 ===")
	alice := &User{
		Username: "gorm_demo_alice",
		Email:    "alice@example.com",
		Password: "hashed_password",
		Balance:  decimal.NewFromFloat(1000.00),
	}
	if err := users.Create(ctx, alice); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("创建成功, ID=%d, 余额=%s\n", alice.ID, alice.Balance.StringFixed(2))

	// 2. 按主键查询单条（First 返回 (User, error)，无需再传 &user）
	fmt.Println("\n=== 2. 查询单条 ===")
	u, err := users.Where("id = ?", alice.ID).First(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("用户: %s (%s)\n", u.Username, u.Email)

	// 3. 条件查询多条（Find 返回 []User, error）
	fmt.Println("\n=== 3. 条件查询 ===")
	enabled, err := users.Where("status = ?", 1).Find(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("找到 %d 个启用用户\n", len(enabled))

	// 4. 分页：先 Count 再分页查询
	fmt.Println("\n=== 4. 分页查询 ===")
	total, err := users.Count(ctx, "id")
	if err != nil {
		log.Fatal(err)
	}
	page, err := users.Order("created_at DESC, id DESC").Offset(0).Limit(2).Find(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("总数: %d, 当前页: %d\n", total, len(page))

	// 5. 更新（Update 返回受影响行数）
	fmt.Println("\n=== 5. 更新 ===")
	rows, err := users.Where("id = ?", alice.ID).Update(ctx, "balance", decimal.NewFromFloat(1500.00))
	if err != nil {
		log.Fatal(err)
	}
	updated, _ := users.Where("id = ?", alice.ID).First(ctx)
	fmt.Printf("更新行数: %d, 更新后余额: %s\n", rows, updated.Balance.StringFixed(2))

	// 6. 软删除（Delete 返回受影响行数）
	fmt.Println("\n=== 6. 软删除 ===")
	if _, err := users.Where("id = ?", alice.ID).Delete(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("用户已软删除")

	// 软删除后默认查不到
	if _, err := users.Where("id = ?", alice.ID).First(ctx); err != nil {
		fmt.Printf("默认查询已查不到: %v\n", err)
	}

	// 包含软删除记录：泛型 API 没有链式 Unscoped() 方法，用 Scopes 在 Statement 层打开
	includeDeleted := func(stmt *gorm.Statement) { stmt.Unscoped = true }
	if du, err := users.Scopes(includeDeleted).Where("id = ?", alice.ID).First(ctx); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Unscoped 可查到, 删除时间: %v\n", du.DeletedAt.Time)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
