// example4-consistency 演示缓存一致性：Cache-Aside 的"先写库再删缓存"
// 仍存在的并发脏读窗口，以及延迟双删的缓解思路。
//
// 本示例不引入消息队列/binlog 订阅（那属于第23章之后的分布式话题），
// 只聚焦单机可复现的时序问题与延迟双删。
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

type ProductCache struct {
	rdb        *redis.Client
	db         store.ProductStore
	ttl        time.Duration
	delayDelay time.Duration // 延迟双删的等待时间
}

func NewProductCache(rdb *redis.Client, db store.ProductStore) *ProductCache {
	return &ProductCache{rdb: rdb, db: db, ttl: 10 * time.Minute, delayDelay: 500 * time.Millisecond}
}

func (c *ProductCache) key(id int64) string {
	return fmt.Sprintf("product:%d", id)
}

func (c *ProductCache) Get(ctx context.Context, id int64) (*store.Product, error) {
	key := c.key(id)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == nil {
		var p store.Product
		_ = json.Unmarshal(data, &p)
		return &p, nil
	}
	if !errors.Is(err, redis.Nil) {
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

// UpdateCacheAside 标准 Cache-Aside 写：先写库，再删缓存。
func (c *ProductCache) UpdateCacheAside(ctx context.Context, p *store.Product) error {
	if err := c.db.Update(ctx, p); err != nil {
		return err
	}
	return c.rdb.Del(ctx, c.key(p.ID)).Err()
}

// UpdateDoubleDelete 延迟双删：写库前后各删一次缓存，第二次延迟执行，
// 覆盖"其他请求在写库间隙把旧值回填缓存"的窗口。
func (c *ProductCache) UpdateDoubleDelete(ctx context.Context, p *store.Product) error {
	key := c.key(p.ID)
	// 第一次删：清掉可能存在的旧缓存。
	c.rdb.Del(ctx, key)
	// 写库。
	if err := c.db.Update(ctx, p); err != nil {
		return err
	}
	// 第二次删：延迟一段时间，删掉写库期间可能被并发读回填的旧值。
	// 生产中放到延迟队列/异步任务里执行，避免阻塞写请求。
	time.AfterFunc(c.delayDelay, func() {
		bg, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := c.rdb.Del(bg, key).Err(); err != nil {
			log.Printf("延迟双删第二次删除失败 key=%s: %v", key, err)
		}
	})
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
	rdb.Del(ctx, "product:2")

	db := store.NewMemStore()
	cache := NewProductCache(rdb, db)

	// 预热缓存
	_, _ = cache.Get(ctx, 2)

	fmt.Println("=== 延迟双删 ===")
	fmt.Println("更新商品价格并触发延迟双删...")
	_ = cache.UpdateDoubleDelete(ctx, &store.Product{ID: 2, Name: "人体工学椅", Price: 79900, Stock: 50})

	p, _ := cache.Get(ctx, 2)
	fmt.Printf("写后立即读: %d 分\n", p.Price)

	fmt.Println("等待延迟删除生效...")
	time.Sleep(700 * time.Millisecond)
	exists, _ := rdb.Exists(ctx, "product:2").Result()
	fmt.Printf("延迟删除后缓存是否存在: %v（0 表示已被第二次删除清掉，下次读将回源最新值）\n", exists == 1)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
