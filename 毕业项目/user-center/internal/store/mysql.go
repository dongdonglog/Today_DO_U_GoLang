package store

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(ctx context.Context, dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(10 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *MySQLStore) CreateUser(ctx context.Context, email, displayName, passwordHash, role string) (*User, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO users (email, display_name, password_hash, role, status)
		VALUES (?, ?, ?, ?, 'active')`,
		email, displayName, passwordHash, role,
	)
	if err != nil {
		if isDuplicate(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, id)
}

func (s *MySQLStore) EnsureAdmin(ctx context.Context, email, displayName, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (email, display_name, password_hash, role, status)
		VALUES (?, ?, ?, 'admin', 'active')
		ON DUPLICATE KEY UPDATE
			role = 'admin',
			status = 'active',
			password_hash = VALUES(password_hash),
			display_name = VALUES(display_name)`,
		email, displayName, passwordHash,
	)
	return err
}

func (s *MySQLStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, display_name, role, status, password_hash, created_at, updated_at
		FROM users WHERE email = ?`,
		email,
	))
}

func (s *MySQLStore) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, display_name, role, status, password_hash, created_at, updated_at
		FROM users WHERE id = ?`,
		id,
	))
}

func (s *MySQLStore) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, display_name, role, status, password_hash, created_at, updated_at
		FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *MySQLStore) SetUserStatus(ctx context.Context, id int64, status string) (*User, error) {
	result, err := s.db.ExecContext(ctx, "UPDATE users SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrNotFound
	}
	return s.GetUserByID(ctx, id)
}

func (s *MySQLStore) SaveRefreshToken(ctx context.Context, tokenID string, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (token_id, user_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)`,
		tokenID, userID, tokenHash, expiresAt,
	)
	return err
}

func (s *MySQLStore) RotateRefreshToken(ctx context.Context, oldTokenID string, userID int64, oldHash string, newTokenID string, newHash string, newExpiresAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var savedHash string
	var expiresAt time.Time
	var revokedAt sql.NullTime
	row := tx.QueryRowContext(ctx, `
		SELECT token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_id = ? AND user_id = ?
		FOR UPDATE`,
		oldTokenID, userID,
	)
	if err := row.Scan(&savedHash, &expiresAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidToken
		}
		return err
	}
	if revokedAt.Valid || time.Now().After(expiresAt) || subtle.ConstantTimeCompare([]byte(savedHash), []byte(oldHash)) != 1 {
		return ErrInvalidToken
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO refresh_tokens (token_id, user_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)`,
		newTokenID, userID, newHash, newExpiresAt,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), replaced_by_token_id = ?
		WHERE token_id = ?`,
		newTokenID, oldTokenID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) RevokeRefreshToken(ctx context.Context, tokenID string, userID int64, tokenHash string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE token_id = ? AND user_id = ? AND token_hash = ?`,
		tokenID, userID, tokenHash,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrInvalidToken
	}
	return nil
}

func scanUser(row *sql.Row) (*User, error) {
	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func isDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
