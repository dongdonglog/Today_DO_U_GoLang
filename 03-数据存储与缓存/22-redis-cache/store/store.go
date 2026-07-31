// Package store 模拟第20章的 GORM + MySQL 数据层。
//
// 为了让缓存示例无需真实 MySQL 也能独立运行，这里用一个带人为延迟和
// 查询计数的内存实现代替数据库。生产代码里把 ProductStore 换成基于
// *gorm.DB 的实现即可，缓存层的逻辑完全不变。
package store

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrNotFound 表示商品在数据库中不存在，对应 gorm.ErrRecordNotFound。
var ErrNotFound = errors.New("product not found")

// Product 商品，字段与第20章的模型保持一致。
type Product struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"` // 单位：分，避免浮点
	Stock int    `json:"stock"`
}

// ProductStore 数据层接口，生产实现由 GORM 提供。
type ProductStore interface {
	GetByID(ctx context.Context, id int64) (*Product, error)
	Update(ctx context.Context, p *Product) error
	// Queries 返回累计的数据库查询次数，用于观察缓存命中率。
	Queries() int64
}

// memStore 内存实现：模拟一次数据库查询约 30ms 的延迟。
type memStore struct {
	mu      sync.RWMutex
	data    map[int64]*Product
	queries int64
	latency time.Duration
}

// NewMemStore 构造一个内存数据层，预置若干商品。
func NewMemStore() ProductStore {
	return &memStore{
		data: map[int64]*Product{
			1: {ID: 1, Name: "机械键盘", Price: 29900, Stock: 100},
			2: {ID: 2, Name: "人体工学椅", Price: 89900, Stock: 50},
			3: {ID: 3, Name: "4K 显示器", Price: 199900, Stock: 30},
		},
		latency: 30 * time.Millisecond,
	}
}

func (s *memStore) GetByID(ctx context.Context, id int64) (*Product, error) {
	atomic.AddInt64(&s.queries, 1)
	// 模拟数据库往返延迟，同时尊重 context 取消。
	select {
	case <-time.After(s.latency):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	// 返回副本，避免调用方改到内部数据。
	cp := *p
	return &cp, nil
}

func (s *memStore) Update(ctx context.Context, p *Product) error {
	atomic.AddInt64(&s.queries, 1)
	select {
	case <-time.After(s.latency):
	case <-ctx.Done():
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[p.ID]; !ok {
		return ErrNotFound
	}
	cp := *p
	s.data[p.ID] = &cp
	return nil
}

func (s *memStore) Queries() int64 {
	return atomic.LoadInt64(&s.queries)
}
