// example4-oversell 用分布式锁解决秒杀超卖问题。
//
// 场景：秒杀活动，商品库存 10，100 个用户并发抢购。
// 不加锁时会超卖；加锁后保证库存不被扣成负数。
//
// 为了让示例可独立运行，库存用 Redis 的 DECR 模拟（真实场景对应 MySQL
// UPDATE ... WHERE stock>0），这里故意用"先 GET 判断再 SET"的非原子操作，
// 让并发问题暴露出来，再用分布式锁修复。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-book/redis-lock/mutex"
	"github.com/redis/go-redis/v9"
)

const stockKey = "seckill:stock:100"
const totalStock = 10

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}

	// ---- 无锁：100 并发抢购，会超卖 ----
	fmt.Println("=== 无锁：100 并发抢购 10 件商品 ===")
	rdb.Set(ctx, stockKey, totalStock, 0)
	var success int64
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 模拟"查库存 > 0 则扣减"的非原子业务逻辑
			n, _ := rdb.Get(ctx, stockKey).Int()
			time.Sleep(5 * time.Millisecond) // 模拟业务耗时，放大并发窗口
			if n > 0 {
				rdb.Set(ctx, stockKey, n-1, 0)
				atomic.AddInt64(&success, 1)
			}
		}()
	}
	wg.Wait()
	remaining, _ := rdb.Get(ctx, stockKey).Int()
	fmt.Printf("成功下单: %d 件，剩余库存: %d（耗时 %v）\n", success, remaining, time.Since(start).Round(time.Millisecond))
	if remaining < 0 || int(success) > totalStock {
		fmt.Println("结果：超卖！库存被扣成负数或下单数超过库存")
	}

	// ---- 加分布式锁：带短暂重试 ----
	fmt.Println("\n=== 加分布式锁 + 短重试：100 并发抢购 10 件商品 ===")
	rdb.Set(ctx, stockKey, totalStock, 0)
	var success2 int64
	start = time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock := mutex.New(rdb, "lock:"+stockKey, mutex.WithTTL(2*time.Second))
			// 抢不到锁短暂自旋重试（更贴近秒杀：等一下再抢），最多等 200ms。
			var acquired bool
			for attempt := 0; attempt < 20; attempt++ {
				if err := lock.Lock(ctx); err == nil {
					acquired = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !acquired {
				return
			}
			defer lock.Unlock(ctx)

			n, _ := rdb.Get(ctx, stockKey).Int()
			time.Sleep(5 * time.Millisecond)
			if n > 0 {
				rdb.Set(ctx, stockKey, n-1, 0)
				atomic.AddInt64(&success2, 1)
			}
		}()
	}
	wg.Wait()
	remaining2, _ := rdb.Get(ctx, stockKey).Int()
	fmt.Printf("成功下单: %d 件，剩余库存: %d（耗时 %v）\n", success2, remaining2, time.Since(start).Round(time.Millisecond))
	if int(success2) == totalStock && remaining2 == 0 {
		fmt.Println("结果：库存刚好用完，没有超卖")
	}

	rdb.Del(ctx, stockKey, "lock:"+stockKey)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
