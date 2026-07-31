// example1-cache-aside 演示 Cache-Aside（旁路缓存）模式。
//
// 读：先查缓存，命中直接返回；未命中回源数据库，再写回缓存。
// 写：先写数据库，再删除缓存（不是更新缓存）。
//
// 启动前请准备 Redis：
//
//	docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine
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

// ProductCache 在数据层之上包一层 Redis 缓存。
type ProductCache struct {
	rdb *redis.Client
	db  store.ProductStore
	ttl time.Duration
}

func NewProductCache(rdb *redis.Client, db store.ProductStore) *ProductCache {
	return &ProductCache{rdb: rdb, db: db, ttl: 10 * time.Minute}
}

func (c *ProductCache) key(id int64) string {
	return fmt.Sprintf("product:%d", id)
}

// Get 读路径：Cache-Aside。
func (c *ProductCache) Get(ctx context.Context, id int64) (*store.Product, error) {
	key := c.key(id)

	// 1. 先查缓存
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == nil {
		var p store.Product
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("反序列化缓存失败: %w", err)
		}
		return &p, nil // 命中
	}
	if !errors.Is(err, redis.Nil) {
		// 注意：Redis 出错时不宜直接失败，生产中应记录日志后降级去查数据库，
		// 否则 Redis 抖动会把流量全部拒绝。这里返回错误是为了让示例行为清晰。
		return nil, fmt.Errorf("读缓存失败: %w", err)
	}

	// 2. 未命中，回源数据库
	p, err := c.db.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. 写回缓存
	buf, _ := json.Marshal(p)
	if err := c.rdb.Set(ctx, key, buf, c.ttl).Err(); err != nil {
		// 写缓存失败不影响本次返回，下次再回源即可。
		log.Printf("写回缓存失败 key=%s: %v", key, err)
	}
	return p, nil
}

// Update 写路径：先写数据库，再删缓存。
func (c *ProductCache) Update(ctx context.Context, p *store.Product) error {
	// 1. 先写数据库
	if err := c.db.Update(ctx, p); err != nil {
		return err
	}
	// 2. 再删缓存（而不是更新缓存，原因见正文 22.5）
	if err := c.rdb.Del(ctx, c.key(p.ID)).Err(); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}
	rdb.Del(ctx, "product:1")

	db := store.NewMemStore()
	cache := NewProductCache(rdb, db)

	fmt.Println("=== 连续读取同一商品 5 次 ===")
	for i := 0; i < 5; i++ {
		start := time.Now()
		p, err := cache.Get(ctx, 1)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("第 %d 次: %s 耗时 %v\n", i+1, p.Name, time.Since(start).Round(time.Millisecond))
	}
	fmt.Printf("数据库查询次数: %d（只有第一次回源）\n", db.Queries())

	fmt.Println("\n=== 更新商品后缓存失效 ===")
	_ = cache.Update(ctx, &store.Product{ID: 1, Name: "机械键盘(改)", Price: 25900, Stock: 80})
	p, _ := cache.Get(ctx, 1)
	fmt.Printf("更新后读到: %s 价格 %d 分\n", p.Name, p.Price)
	fmt.Printf("数据库查询次数: %d（更新+回源各一次）\n", db.Queries())
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
