// example2-penetration 演示缓存穿透的两种防御：空值缓存 + 布隆过滤器。
//
// 缓存穿透：查询一个数据库里根本不存在的 key，缓存永远不命中，
// 每次都打到数据库。常见于恶意用 -1、随机 ID 刷接口。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-book/redis-cache/store"
	"github.com/redis/go-redis/v9"
)

// nullValue 是空值缓存的占位符：数据库查不到时，缓存一个短 TTL 的空标记，
// 挡住对同一不存在 key 的重复穿透。
const nullValue = "__NULL__"

type ProductCache struct {
	rdb      *redis.Client
	db       store.ProductStore
	bloom    *BloomFilter
	ttl      time.Duration
	nullTTL  time.Duration // 空值缓存用更短的 TTL，避免长期占内存 + 误挡新增数据
	useBloom bool
}

func NewProductCache(rdb *redis.Client, db store.ProductStore, bloom *BloomFilter) *ProductCache {
	return &ProductCache{
		rdb:      rdb,
		db:       db,
		bloom:    bloom,
		ttl:      10 * time.Minute,
		nullTTL:  1 * time.Minute,
		useBloom: bloom != nil,
	}
}

func (c *ProductCache) key(id int64) string {
	return fmt.Sprintf("product:%d", id)
}

// Get 带穿透防御的读路径。
func (c *ProductCache) Get(ctx context.Context, id int64) (*store.Product, error) {
	// 0. 布隆过滤器前置拦截：一定不存在的 id 直接拒绝，连缓存都不查。
	if c.useBloom {
		exists, err := c.bloom.MightContain(ctx, id)
		if err != nil {
			log.Printf("布隆过滤器查询失败，降级放行: %v", err)
		} else if !exists {
			return nil, store.ErrNotFound
		}
	}

	key := c.key(id)

	// 1. 查缓存
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == nil {
		if string(data) == nullValue {
			return nil, store.ErrNotFound // 命中空值缓存
		}
		var p store.Product
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("反序列化缓存失败: %w", err)
		}
		return &p, nil
	}
	if !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("读缓存失败: %w", err)
	}

	// 2. 回源
	p, err := c.db.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		// 关键：数据库也查不到时，缓存一个短 TTL 空值，挡住后续穿透。
		c.rdb.Set(ctx, key, nullValue, c.nullTTL)
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	buf, _ := json.Marshal(p)
	c.rdb.Set(ctx, key, buf, c.ttl)
	return p, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}

	db := store.NewMemStore()

	// --- 方案一：空值缓存 ---
	fmt.Println("=== 方案一：空值缓存 ===")
	rdb.Del(ctx, "product:999")
	cache := NewProductCache(rdb, db, nil)
	for i := 0; i < 3; i++ {
		_, err := cache.Get(ctx, 999) // 不存在的商品
		fmt.Printf("第 %d 次查询 id=999: %v\n", i+1, err)
	}
	fmt.Printf("数据库查询次数: %d（只穿透了 1 次，之后被空值缓存挡住）\n", db.Queries())

	// --- 方案二：布隆过滤器 ---
	fmt.Println("\n=== 方案二：布隆过滤器 ===")
	bloomKey := "bloom:products"
	// 清掉方案一留下的空值缓存与方案本身的布隆 key，确保独立运行两段输出一致。
	rdb.Del(ctx, bloomKey, "product:1", "product:999", "product:12345")
	bloom := NewBloomFilter(rdb, bloomKey, 10000, 3)
	// 预热：把所有存在的商品 id 灌入布隆过滤器（生产中在启动或写入时维护）。
	for _, id := range []int64{1, 2, 3} {
		_ = bloom.Add(ctx, id)
	}
	db2 := store.NewMemStore()
	cache2 := NewProductCache(rdb, db2, bloom)
	for _, id := range []int64{1, 999, 12345} {
		_, err := cache2.Get(ctx, id)
		fmt.Printf("查询 id=%d: %v\n", id, err)
	}
	fmt.Printf("数据库查询次数: %d（只有存在的 id=1 回源，不存在的被布隆挡在缓存之前）\n", db2.Queries())
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
