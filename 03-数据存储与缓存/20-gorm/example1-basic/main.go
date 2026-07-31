package main

import (
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

	fmt.Println("连接成功")

	// 1. 查询单条
	fmt.Println("\n=== 1. 查询单条 ===")
	var user User
	result := db.First(&user, 1)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	fmt.Printf("用户: %+v\n", user)

	// 2. 条件查询
	fmt.Println("\n=== 2. 条件查询 ===")
	var users []User
	if err := db.Where("status = ?", 1).Find(&users).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("找到 %d 个启用用户\n", len(users))
	for _, u := range users {
		fmt.Printf("  - %s (%s)\n", u.Username, u.Email)
	}

	// 3. 分页查询
	fmt.Println("\n=== 3. 分页查询 ===")
	var total int64
	var pageUsers []User
	if err := db.Model(&User{}).Count(&total).Error; err != nil {
		log.Fatal(err)
	}
	if err := db.Order("created_at DESC, id DESC").Offset(0).Limit(2).Find(&pageUsers).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("总数: %d, 当前页: %d\n", total, len(pageUsers))

	// 4. 更新
	fmt.Println("\n=== 4. 更新 ===")
	if err := db.Model(&user).Update("balance", decimal.NewFromFloat(1500.00)).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("更新后余额: %s\n", user.Balance.StringFixed(2))

	// 5. 软删除
	fmt.Println("\n=== 5. 软删除 ===")
	if err := db.Delete(&user).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Println("用户已软删除")

	// 验证软删除
	var deletedUser User
	err = db.First(&deletedUser, user.ID).Error
	fmt.Printf("删除后查询错误: %v\n", err)

	// 查询包含软删除
	if err := db.Unscoped().First(&deletedUser, user.ID).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("包含软删除: %+v\n", deletedUser)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
