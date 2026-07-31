package main

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/go-book/redis-cache/store"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// logicalValue 是逻辑过期缓存的存储结构：数据本身永不设物理 TTL，
// 而是在值里带一个逻辑过期时间戳。读到已逻辑过期的数据时，
// 先返回旧值保证可用，再由后台异步重建，从根本上避免击穿。
type logicalValue struct {
	Product  *store.Product `json:"product"`
	ExpireAt int64          `json:"expire_at"` // 逻辑过期时间（Unix 毫秒）
}

// LogicalCache 逻辑过期缓存。
type LogicalCache struct {
	rdb        *redis.Client
	db         store.ProductStore
	logicalTTL time.Duration
	sf         singleflight.Group
}

func NewLogicalCache(rdb *redis.Client, db store.ProductStore) *LogicalCache {
	return &LogicalCache{rdb: rdb, db: db, logicalTTL: time.Second}
}

func (c *LogicalCache) key(id int64) string {
	return "product:logical:" + strconv.FormatInt(id, 10)
}

// Warmup 预热：把数据写入缓存，不设物理过期，只带逻辑过期时间。
func (c *LogicalCache) Warmup(ctx context.Context, id int64) error {
	p, err := c.db.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return c.set(ctx, id, p)
}

func (c *LogicalCache) set(ctx context.Context, id int64, p *store.Product) error {
	lv := logicalValue{Product: p, ExpireAt: time.Now().Add(c.logicalTTL).UnixMilli()}
	buf, _ := json.Marshal(lv)
	// 物理 TTL=0：永不过期，靠逻辑时间戳判断是否需要重建。
	return c.rdb.Set(ctx, c.key(id), buf, 0).Err()
}

// Get 读逻辑过期缓存：
//   - 未逻辑过期：直接返回。
//   - 已逻辑过期：立即返回旧值，同时用 singleflight 触发一次后台异步重建。
func (c *LogicalCache) Get(ctx context.Context, id int64) (*store.Product, error) {
	data, err := c.rdb.Get(ctx, c.key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		// 缓存不存在（未预热），同步回源一次。
		p, err := c.db.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		_ = c.set(ctx, id, p)
		return p, nil
	}
	if err != nil {
		return nil, err
	}

	var lv logicalValue
	if err := json.Unmarshal(data, &lv); err != nil {
		return nil, err
	}

	if time.Now().UnixMilli() < lv.ExpireAt {
		return lv.Product, nil // 未逻辑过期
	}

	// 已逻辑过期：先返回旧值，后台异步重建（同一 key 只触发一次）。
	go c.rebuild(id)
	return lv.Product, nil
}

func (c *LogicalCache) rebuild(id int64) {
	c.sf.Do(c.key(id), func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		p, err := c.db.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return nil, c.set(ctx, id, p)
	})
}
