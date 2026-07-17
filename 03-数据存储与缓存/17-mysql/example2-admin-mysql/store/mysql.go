package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-book/mysql/example2-admin-mysql/model"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLStore MySQL 存储
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore 创建 MySQL 存储
func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// 设置连接池
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(10 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return &MySQLStore{db: db}, nil
}

// Close 关闭连接
func (s *MySQLStore) Close() error {
	return s.db.Close()
}

type listQuery struct {
	CountSQL string
	ListSQL  string
	Args     []interface{}
	ListArgs []interface{}
	Page     int
	Size     int
}

func buildListQuery(req *model.ListUsersRequest) listQuery {
	// 构建 WHERE 条件
	where := []string{"deleted_at IS NULL"}
	args := []interface{}{}

	if req.Email != "" {
		where = append(where, "active_email = ?")
		args = append(args, req.Email)
	}

	if req.Status != nil {
		where = append(where, "status = ?")
		args = append(args, *req.Status)
	}

	whereStr := strings.Join(where, " AND ")

	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.Size
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", whereStr)
	listSQL := fmt.Sprintf(
		"SELECT id, username, email, phone, status, created_at, updated_at FROM users WHERE %s ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		whereStr,
	)

	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, size, offset)

	return listQuery{
		CountSQL: countSQL,
		ListSQL:  listSQL,
		Args:     args,
		ListArgs: listArgs,
		Page:     page,
		Size:     size,
	}
}

// List 查询用户列表
func (s *MySQLStore) List(ctx context.Context, req *model.ListUsersRequest) ([]*model.User, int, error) {
	query := buildListQuery(req)

	// 查询总数
	var total int
	err := s.db.QueryRowContext(ctx, query.CountSQL, query.Args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	rows, err := s.db.QueryContext(ctx, query.ListSQL, query.ListArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		var user model.User
		var phone sql.NullString
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &phone, &user.Status, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		user.Phone = nullStringPtr(phone)
		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetByID 根据 ID 查询用户
func (s *MySQLStore) GetByID(ctx context.Context, id int64) (*model.User, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, username, email, phone, status, created_at, updated_at FROM users WHERE id = ? AND deleted_at IS NULL",
		id,
	)

	var user model.User
	var phone sql.NullString
	err := row.Scan(&user.ID, &user.Username, &user.Email, &phone, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %d", id)
		}
		return nil, err
	}
	user.Phone = nullStringPtr(phone)

	return &user, nil
}

// GetByEmail 根据邮箱查询用户
func (s *MySQLStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, username, email, phone, status, created_at, updated_at FROM users WHERE active_email = ?",
		email,
	)

	var user model.User
	var phone sql.NullString
	err := row.Scan(&user.ID, &user.Username, &user.Email, &phone, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", email)
		}
		return nil, err
	}
	user.Phone = nullStringPtr(phone)

	return &user, nil
}

// Create 创建用户
func (s *MySQLStore) Create(ctx context.Context, req *model.CreateUserRequest, hashedPassword string) (*model.User, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO users (username, email, password_hash, phone, status) VALUES (?, ?, ?, ?, 1)",
		req.Username, req.Email, hashedPassword, nullableString(req.Phone),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Update 更新用户
func (s *MySQLStore) Update(ctx context.Context, id int64, req *model.UpdateUserRequest) (*model.User, error) {
	set := []string{}
	args := []interface{}{}

	if req.Phone != nil {
		set = append(set, "phone = ?")
		args = append(args, nullableString(*req.Phone))
	}

	if req.Status != nil {
		set = append(set, "status = ?")
		args = append(args, *req.Status)
	}

	if len(set) == 0 {
		return s.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ? AND deleted_at IS NULL", strings.Join(set, ", "))

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, fmt.Errorf("user not found: %d", id)
	}

	return s.GetByID(ctx, id)
}

// Delete 删除用户（软删除）
func (s *MySQLStore) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, "UPDATE users SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %d", id)
	}

	return nil
}

func nullableString(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
