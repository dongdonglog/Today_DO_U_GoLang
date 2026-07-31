package main

import (
	"context"
	"errors"
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
	List(ctx context.Context, req ListUsersRequest) ([]*User, int64, error)
	DeductBalance(ctx context.Context, userID uint, amount decimal.Decimal) error
}

type ListUsersRequest struct {
	Page int
	Size int
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Preload("Orders").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) List(ctx context.Context, req ListUsersRequest) ([]*User, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Size > 100 {
		req.Size = 100
	}

	var users []*User
	var total int64

	query := r.db.WithContext(ctx).Model(&User{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.Size
	err := query.Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&users).Error

	return users, total, err
}

// DeductBalance 扣减余额（乐观锁）
func (r *userRepository) DeductBalance(ctx context.Context, userID uint, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("扣减金额必须大于 0")
	}

	for i := 0; i < 3; i++ {
		var user User
		if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
			return err
		}

		if user.Balance.Cmp(amount) < 0 {
			return errors.New("余额不足")
		}

		result := r.db.WithContext(ctx).
			Model(&User{}).
			Where("id = ? AND version = ?", userID, user.Version).
			Updates(map[string]interface{}{
				"balance": gorm.Expr("balance - ?", amount),
				"version": gorm.Expr("version + 1"),
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 {
			return nil
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
	dsn := getenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/go_book_gorm?charset=utf8mb4&parseTime=true")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("连接失败:", err)
	}

	fmt.Println("连接成功")

	repo := NewUserRepository(db)
	ctx := context.Background()

	// 1. 查询用户（带订单）
	fmt.Println("\n=== 1. 查询用户（带订单）===")
	user, err := repo.GetByID(ctx, 1)
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
		fmt.Printf("  - %s\n", u.Username)
	}

	// 3. 乐观锁扣减余额
	fmt.Println("\n=== 3. 乐观锁扣减余额 ===")
	err = repo.DeductBalance(ctx, 1, decimal.NewFromFloat(100.00))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("扣减成功")

	// 验证余额
	user, err = repo.GetByID(ctx, 1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("当前余额: %s\n", user.Balance.StringFixed(2))

	// 4. 事务示例
	fmt.Println("\n=== 4. 事务示例 ===")
	err = db.Transaction(func(tx *gorm.DB) error {
		// 扣钱
		if err := tx.Model(&User{}).Where("id = ?", 1).
			Update("balance", gorm.Expr("balance - ?", 50)).Error; err != nil {
			return err
		}

		// 加钱
		if err := tx.Model(&User{}).Where("id = ?", 2).
			Update("balance", gorm.Expr("balance + ?", 50)).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Fatal("事务失败:", err)
	}
	fmt.Println("转账成功")
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
