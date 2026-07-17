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

// User 用户模型
type User struct {
	ID    int64
	Name  string
	Email string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接 MySQL
	// 格式：user:password@tcp(host:port)/
	dsn := getenv("MYSQL_ROOT_DSN", "root:root@tcp(localhost:3306)/?charset=utf8mb4&parseTime=true")
	rootDB, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer rootDB.Close()

	if _, err := rootDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS go_book_basic DEFAULT CHARACTER SET utf8mb4"); err != nil {
		log.Fatal("Failed to create database:", err)
	}

	db, err := sql.Open("mysql", getenv("MYSQL_BASIC_DSN", "root:root@tcp(localhost:3306)/go_book_basic?charset=utf8mb4&parseTime=true"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Failed to connect to MySQL:", err)
	}
	fmt.Println("Connected to MySQL successfully!")

	// 创建表
	createTable(ctx, db)

	// CRUD 操作
	// 创建
	userID := createUser(ctx, db, fmt.Sprintf("alice-%d@example.com", time.Now().UnixNano()))
	fmt.Printf("Created user with ID: %d\n", userID)

	// 查询
	user := getUser(ctx, db, userID)
	fmt.Printf("Got user: %+v\n", user)

	// 更新
	updateUser(ctx, db, userID, "Alice Updated")
	user = getUser(ctx, db, userID)
	fmt.Printf("Updated user: %+v\n", user)

	// 删除
	deleteUser(ctx, db, userID)
	fmt.Println("Deleted user")
}

func createTable(ctx context.Context, db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		name VARCHAR(50) NOT NULL,
		email VARCHAR(100) NOT NULL UNIQUE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	fmt.Println("Table created successfully!")
}

func createUser(ctx context.Context, db *sql.DB, email string) int64 {
	result, err := db.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", email)
	if err != nil {
		log.Fatal("Failed to create user:", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Fatal("Failed to get insert id:", err)
	}
	return id
}

func getUser(ctx context.Context, db *sql.DB, id int64) *User {
	row := db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = ?", id)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Fatal("Failed to get user:", err)
	}

	return &user
}

func updateUser(ctx context.Context, db *sql.DB, id int64, name string) {
	_, err := db.ExecContext(ctx, "UPDATE users SET name = ? WHERE id = ?", name, id)
	if err != nil {
		log.Fatal("Failed to update user:", err)
	}
}

func deleteUser(ctx context.Context, db *sql.DB, id int64) {
	_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		log.Fatal("Failed to delete user:", err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
