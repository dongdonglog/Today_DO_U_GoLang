package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrConflict = errors.New("conflict")
	ErrNoStock  = errors.New("not enough stock")
	ErrNotFound = errors.New("not found")
)

type Store struct {
	db *sql.DB
}

type Order struct {
	OrderNo     string    `json:"order_no"`
	UserID      int64     `json:"user_id"`
	SkuID       int64     `json:"sku_id"`
	Quantity    int64     `json:"quantity"`
	AmountCents int64     `json:"amount_cents"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateOrderInput struct {
	IdempotencyKey string
	UserID         int64
	SkuID          int64
	Quantity       int64
}

type OutboxEvent struct {
	ID          int64
	EventID     string
	EventType   string
	AggregateID string
	Payload     []byte
}

type OrderEvent struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	OrderNo     string `json:"order_no"`
	UserID      int64  `json:"user_id"`
	SkuID       int64  `json:"sku_id"`
	Quantity    int64  `json:"quantity"`
	AmountCents int64  `json:"amount_cents"`
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(40)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(10 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) GetStock(ctx context.Context, skuID int64) (int64, error) {
	var stock int64
	err := s.db.QueryRowContext(ctx, "SELECT stock FROM products WHERE sku_id = ?", skuID).Scan(&stock)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return stock, err
}

func (s *Store) GetOrderByNo(ctx context.Context, orderNo string) (*Order, error) {
	return scanOrder(s.db.QueryRowContext(ctx, `
		SELECT order_no, user_id, sku_id, quantity, amount_cents, status, created_at
		FROM orders WHERE order_no = ?`, orderNo))
}

func (s *Store) GetOrderByIdempotencyKey(ctx context.Context, key string) (*Order, error) {
	return scanOrder(s.db.QueryRowContext(ctx, `
		SELECT order_no, user_id, sku_id, quantity, amount_cents, status, created_at
		FROM orders WHERE idempotency_key = ?`, key))
}

func (s *Store) CreateOrder(ctx context.Context, input CreateOrderInput) (*Order, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	var price int64
	if err := tx.QueryRowContext(ctx, "SELECT price_cents FROM products WHERE sku_id = ?", input.SkuID).Scan(&price); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, ErrNotFound
		}
		return nil, false, err
	}

	orderNo := fmt.Sprintf("NO-%d", time.Now().UnixNano())
	amount := price * input.Quantity
	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (order_no, idempotency_key, user_id, sku_id, quantity, amount_cents, status)
		VALUES (?, ?, ?, ?, ?, ?, 'created')`,
		orderNo, input.IdempotencyKey, input.UserID, input.SkuID, input.Quantity, amount,
	)
	if err != nil {
		if isDuplicate(err) {
			order, getErr := s.GetOrderByIdempotencyKey(ctx, input.IdempotencyKey)
			return order, false, getErr
		}
		return nil, false, err
	}

	res, err := tx.ExecContext(ctx,
		"UPDATE products SET stock = stock - ? WHERE sku_id = ? AND stock >= ?",
		input.Quantity, input.SkuID, input.Quantity,
	)
	if err != nil {
		return nil, false, err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	if changed == 0 {
		return nil, false, ErrNoStock
	}

	eventID := fmt.Sprintf("evt-%s-created", orderNo)
	payload, _ := json.Marshal(OrderEvent{
		EventID:     eventID,
		EventType:   "order.created",
		OrderNo:     orderNo,
		UserID:      input.UserID,
		SkuID:       input.SkuID,
		Quantity:    input.Quantity,
		AmountCents: amount,
	})
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO outbox (event_id, event_type, aggregate_id, payload)
		VALUES (?, 'order.created', ?, ?)`,
		eventID, orderNo, string(payload),
	); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	order, err := s.GetOrderByNo(ctx, orderNo)
	return order, true, err
}

func (s *Store) FetchOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_id, event_type, aggregate_id, payload
		FROM outbox
		WHERE sent = 0 AND next_retry_at <= NOW()
		ORDER BY id ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0)
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(&event.ID, &event.EventID, &event.EventType, &event.AggregateID, &event.Payload); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) MarkOutboxSent(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE outbox SET sent = 1 WHERE id = ?", id)
	return err
}

func (s *Store) MarkOutboxRetry(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE outbox
		SET retry_count = retry_count + 1,
		    next_retry_at = DATE_ADD(NOW(), INTERVAL LEAST(retry_count + 1, 30) SECOND)
		WHERE id = ?`, id)
	return err
}

func (s *Store) MarkOrderPaid(ctx context.Context, event OrderEvent, handler string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		"INSERT IGNORE INTO processed_events (event_id, handler) VALUES (?, ?)",
		event.EventID, handler,
	)
	if err != nil {
		return err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		return tx.Commit()
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE orders SET status = 'paid' WHERE order_no = ? AND status = 'created'",
		event.OrderNo,
	); err != nil {
		return err
	}

	paidID := fmt.Sprintf("evt-%s-paid", event.OrderNo)
	event.EventID = paidID
	event.EventType = "order.paid"
	payload, _ := json.Marshal(event)
	if _, err := tx.ExecContext(ctx, `
		INSERT IGNORE INTO outbox (event_id, event_type, aggregate_id, payload)
		VALUES (?, 'order.paid', ?, ?)`,
		paidID, event.OrderNo, string(payload),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func scanOrder(row *sql.Row) (*Order, error) {
	var order Order
	if err := row.Scan(&order.OrderNo, &order.UserID, &order.SkuID, &order.Quantity, &order.AmountCents, &order.Status, &order.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

func isDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
