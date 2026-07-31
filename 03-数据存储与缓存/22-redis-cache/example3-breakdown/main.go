// example3-breakdown 演示缓存击穿的两种防御：singleflight 合并回源 + 逻辑过期。
//
// 缓存击穿：某个热点 key 过期的瞬间，大量并发请求同时未命中，
// 全部涌向数据库回源，把数据库压垮。区别于雪崩（大批不同 key 同时过期）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-book/redis-cache/store"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type ProductCache struct {
	rdb *redis.Client
	db  store.ProductStore
	ttl time.Duration
	sf  singleflight.Group
}

func NewProductCache(rdb *redis.Client, db store.ProductStore) *ProductCache {
	return &ProductCache{rdb: rdb, db: db, ttl: 10 * time.Minute}
}

func (c *ProductCache) key(id int64) string {
	return fmt.Sprintf("product:%d", id)
}

// GetNaive 无保护的读：并发未命中时会同时回源，用于对比。
func (c *ProductCache) GetNaive(ctx context.Context, id int64) (*store.Product, error) {
	key := c.key(id)
	if data, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var p store.Product
		_ = json.Unmarshal(data, &p)
		return &p, nil
	} else if !errors.Is(err, redis.Nil) {
		return nil, err
	}
	p, err := c.db.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	buf, _ := json.Marshal(p)
	c.rdb.Set(ctx, key, buf, c.ttl)
	return p, nil
}

// GetWithSingleflight 用 singleflight 合并同一 key 的并发回源：
// 同一时刻对同一 key，只有一个 goroutine 真正查数据库，其余共享结果。
func (c *ProductCache) GetWithSingleflight(ctx context.Context, id int64) (*store.Product, error) {
	key := c.key(id)
	if data, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var p store.Product
		_ = json.Unmarshal(data, &p)
		return &p, nil
	} else if !errors.Is(err, redis.Nil) {
		return nil, err
	}

	// key 相同的并发调用会被合并，Do 只执行一次回源。
	v, err, _ := c.sf.Do(key, func() (interface{}, error) {
		p, err := c.db.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		buf, _ := json.Marshal(p)
		c.rdb.Set(ctx, key, buf, c.ttl)
		return p, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*store.Product), nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}

	const concurrency = 100

	// --- 对比一：无保护，热点 key 过期瞬间 100 并发 ---
	fmt.Println("=== 无保护：100 并发同时未命中 ===")
	rdb.Del(ctx, "product:1")
	db1 := store.NewMemStore()
	cache1 := NewProductCache(rdb, db1)
	runConcurrent(ctx, concurrency, cache1.GetNaive)
	fmt.Printf("数据库查询次数: %d（大量并发在缓存写回前穿透到数据库）\n", db1.Queries())

	// --- 对比二：singleflight 合并回源 ---
	fmt.Println("\n=== singleflight：100 并发同时未命中 ===")
	rdb.Del(ctx, "product:1")
	db2 := store.NewMemStore()
	cache2 := NewProductCache(rdb, db2)
	runConcurrent(ctx, concurrency, cache2.GetWithSingleflight)
	fmt.Printf("数据库查询次数: %d（并发被合并为极少数几次回源）\n", db2.Queries())

	// --- 对比三：逻辑过期（永不物理过期，异步重建）---
	fmt.Println("\n=== 逻辑过期：热点数据永不物理过期 ===")
	rdb.Del(ctx, "product:1")
	db3 := store.NewMemStore()
	lc := NewLogicalCache(rdb, db3)
	_ = lc.Warmup(ctx, 1)               // 预热写入带逻辑过期时间的缓存
	time.Sleep(1100 * time.Millisecond) // 等逻辑过期时间到
	runConcurrentLogical(ctx, concurrency, lc)
	time.Sleep(200 * time.Millisecond) // 等异步重建完成
	fmt.Printf("数据库查询次数: %d（1 次预热 + 100 并发下后台仅触发 1 次异步重建，请求都拿旧值秒回）\n", db3.Queries())
}

func runConcurrent(ctx context.Context, n int, fn func(context.Context, int64) (*store.Product, error)) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = fn(ctx, 1)
		}()
	}
	wg.Wait()
}

func runConcurrentLogical(ctx context.Context, n int, lc *LogicalCache) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = lc.Get(ctx, 1)
		}()
	}
	wg.Wait()
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
