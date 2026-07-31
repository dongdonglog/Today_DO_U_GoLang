// example3-mutex 演示生产可用的互斥锁：原子加锁、Lua 释放、看门狗续约。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-book/redis-lock/mutex"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}

	const key = "lock:order:1003"
	rdb.Del(ctx, key)

	// ---- 1. 基本加解锁 ----
	fmt.Println("=== 基本加锁/释放 ===")
	lock := mutex.New(rdb, key, mutex.WithTTL(5*time.Second))
	if err := lock.Lock(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("拿到锁")

	// 第二把锁立刻拿不到
	lock2 := mutex.New(rdb, key, mutex.WithTTL(5*time.Second))
	err := lock2.Lock(ctx)
	fmt.Printf("其他客户端加锁：%v（预期 lock not acquired）\n", err)

	if err := lock.Unlock(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("释放锁")

	// 释放后第二个客户端能拿到
	if err := lock2.Lock(ctx); err == nil {
		fmt.Println("释放后 lock2 成功拿到锁")
		_ = lock2.Unlock(ctx)
	}

	// ---- 2. 看门狗续约 ----
	fmt.Println("\n=== 看门狗自动续约 ===")
	longKey := "lock:long-task"
	rdb.Del(ctx, longKey)
	lock3 := mutex.New(rdb, longKey, mutex.WithTTL(4*time.Second)) // TTL=4s
	if err := lock3.Lock(ctx); err != nil {
		log.Fatal(err)
	}
	stop, errCh := lock3.RunWatchdog(ctx)

	// 业务实际跑 10s，远超 TTL
	fmt.Println("业务开始执行（10s），看门狗每 ~1.3s 续约一次")
	done := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Second)
		close(done)
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
loop:
	for {
		select {
		case <-done:
			break loop
		case <-ticker.C:
			ttl, _ := rdb.TTL(ctx, longKey).Result()
			fmt.Printf("  业务运行中，锁 TTL ≈ %v（持续续约，不会过期）\n", ttl.Round(time.Second))
		case e := <-errCh:
			log.Fatalf("续约异常: %v", e)
		}
	}
	stop()
	_ = lock3.Unlock(ctx)
	fmt.Println("业务结束，锁已释放")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
