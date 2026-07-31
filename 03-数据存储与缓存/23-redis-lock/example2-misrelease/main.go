// example2-misrelease 演示没有持有者标识时的误释放问题。
//
// A 拿到锁后业务执行超时，锁自动过期；B 拿到同一把锁；
// 这时 A 执行完去 DEL，会把 B 的锁删掉。必须用唯一 token + Lua 原子校验。
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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}

	const key = "lock:order:1002"
	rdb.Del(ctx, key)

	// 给锁设个短 TTL，模拟 A 业务超时导致锁过期。
	const shortTTL = 2 * time.Second

	// ---- 错误释放：直接 DEL，不校验持有者 ----
	fmt.Println("=== 错误释放：不校验持有者直接 DEL ===")
	rdb.SetNX(ctx, key, "client-a", shortTTL)
	fmt.Println("A 拿到锁，开始执行业务...")
	time.Sleep(shortTTL + 100*time.Millisecond) // A 业务"超时"，锁自动过期

	ok, _ := rdb.SetNX(ctx, key, "client-b", 10*time.Second).Result()
	fmt.Printf("锁已过期，B 拿到锁: %v\n", ok)

	// A 终于执行完，直接 DEL：
	rdb.Del(ctx, key)
	fmt.Println("A 执行完 DEL key，把 B 的锁删了！")

	held, _ := rdb.Exists(ctx, key).Result()
	fmt.Printf("B 的锁还在吗？存在=%v（0 表示被误删）\n", held == 1)

	// ---- 正确释放：Lua 校验 owner == 自己再 DEL ----
	fmt.Println("\n=== 正确释放：Lua 校验持有者 ===")
	rdb.Del(ctx, key)
	const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end
`
	rdb.SetNX(ctx, key, "token-a", shortTTL)
	time.Sleep(shortTTL + 100*time.Millisecond)
	rdb.SetNX(ctx, key, "token-b", 10*time.Second)

	// A 试图用自己的 token 释放
	res, _ := rdb.Eval(ctx, unlockScript, []string{key}, "token-a").Result()
	fmt.Printf("A 用旧 token 释放：n=%d（0=没删到，B 的锁安全）\n", res)
	held2, _ := rdb.Exists(ctx, key).Result()
	fmt.Printf("B 的锁还在吗？存在=%v（1 表示安全）\n", held2 == 1)

	// B 用自己的 token 释放
	res2, _ := rdb.Eval(ctx, unlockScript, []string{key}, "token-b").Result()
	fmt.Printf("B 用自己的 token 释放：n=%d（1=删除成功）\n", res2)

	rdb.Del(ctx, key)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
