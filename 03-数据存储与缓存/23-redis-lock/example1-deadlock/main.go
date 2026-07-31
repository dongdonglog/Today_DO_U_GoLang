// example1-deadlock 演示朴素 SETNX 的死锁问题。
//
// 正确的锁必须在加锁时原子地设置过期时间。
// 如果 SETNX 和 EXPIRE 分成两步，中间进程崩溃，锁就永远不会释放。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}

	const key = "lock:order:1001"
	rdb.Del(ctx, key)

	// ---- 错误写法：两步操作（SETNX + EXPIRE），中间"崩溃" ----
	fmt.Println("=== 错误写法：SETNX 后、EXPIRE 前进程崩溃 ===")
	ok, _ := rdb.SetNX(ctx, key, "client-a", 0).Result() // 第1步：拿到锁，永不过期
	fmt.Printf("SETNX 成功: %v\n", ok)

	// 假设在这里进程崩溃（panic / OOM / SIGKILL），EXPIRE 永远执行不到：
	// rdb.Expire(ctx, key, 10*time.Second)  ← 没执行
	fmt.Println("进程崩溃（模拟：跳过 EXPIRE）")

	// ---- 另一个客户端试图加锁，永远拿不到 ----
	ttl, _ := rdb.TTL(ctx, key).Result()
	fmt.Printf("当前锁 TTL: %v（-1 表示永不过期）\n", ttl)
	ok2, _ := rdb.SetNX(ctx, key, "client-b", 10*time.Second).Result()
	fmt.Printf("client-b 试图加锁: %v（false=拿不到，锁已泄漏）\n", ok2)

	fmt.Println("\n=== 正确写法：SET NX PX 原子加锁 ===")
	rdb.Del(ctx, key)
	// 一次性原子地：不存在才设置 + 同时给过期时间。中间崩溃也会自动过期。
	ok3, _ := rdb.SetNX(ctx, key, "client-a", 10*time.Second).Result()
	fmt.Printf("SETNX+TTL 原子加锁成功: %v\n", ok3)
	ttl2, _ := rdb.TTL(ctx, key).Result()
	fmt.Printf("当前锁 TTL: %v（崩溃后也会自动释放）\n", ttl2.Round(time.Second))

	// 清理
	rdb.Del(ctx, key)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
