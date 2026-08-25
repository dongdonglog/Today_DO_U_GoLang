package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

// 分布式定时任务:每 3 秒聚合一次成交数据。
//
//	go run . -name=worker-a   # 终端 A
//	go run . -name=worker-b   # 终端 B
//
// 观察:同一个周期只有一个 worker 执行——靠 Redis 分布式锁(第23章的 SET NX PX)。
// 没抢到的实例这个周期直接跳过,而不是排队执行(定时任务不需要背压)。
//
// 关键设计:
//   - 锁的 TTL < 任务间隔,且持锁者执行完主动释放:防止实例崩溃后锁泄漏导致任务停摆
//   - 用「时间片」做锁的 key(如 minute bucket),而不是固定 key+自旋:
//     同一分钟内即使有实例崩溃释放了锁,其他实例补跑的也是同一份工作语义
func main() {
	name := "worker-a"
	for _, a := range os.Args[1:] {
		if len(a) > 6 && a[:6] == "-name=" {
			name = a[6:]
		}
	}

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	fmt.Printf("[%s] 定时任务启动,每 3s 一个周期\n", name)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] 退出\n", name)
			return
		case now := <-ticker.C:
			runOnce(ctx, rdb, name, now)
		}
	}
}

// runOnce 尝试获取本时间片的执行权并执行任务
func runOnce(ctx context.Context, rdb *redis.Client, name string, now time.Time) {
	// 时间片 key:按秒对齐,所有实例对"同一周期"竞争同一把锁
	slot := now.Truncate(time.Second).Unix()
	lockKey := fmt.Sprintf("cron:aggregate:%d", slot)

	token := fmt.Sprintf("%s-%d", name, rand.Int63())

	// SET NX PX:原子加锁,TTL 兜底(持锁实例崩溃,锁也会自动过期)
	ok, err := rdb.SetNX(ctx, lockKey, token, 10*time.Second).Result()
	if err != nil {
		log.Printf("[%s] 加锁出错: %v", name, err)
		return
	}
	if !ok {
		fmt.Printf("[%s] 本周期由其他实例执行,跳过\n", name)
		return
	}
	defer releaseLock(ctx, rdb, lockKey, token)

	start := time.Now()
	time.Sleep(300 * time.Millisecond) // 模拟聚合查询耗时

	// 幂等保障的第二道防线:把结果写到带时间片 key 的 Redis 里,
	// 即使极端情况下重复执行,写入也是覆盖而非累加
	rdb.Set(ctx, fmt.Sprintf("report:%d", slot), "done", time.Hour)

	fmt.Printf("[%s] 执行聚合完成 耗时=%v (slot=%d)\n",
		name, time.Since(start).Round(time.Millisecond), slot)
}

// releaseLock 只释放自己持有的锁(Lua 校验 token,见第23章)
func releaseLock(ctx context.Context, rdb *redis.Client, key, token string) {
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	if _, err := rdb.Eval(ctx, script, []string{key}, token).Result(); err != nil {
		log.Printf("[%s] 释放锁失败(TTL 会兜底): %v", name_of(token), err)
	}
}

func name_of(token string) string {
	if i := strings.Index(token, "-"); i > 0 {
		return token[:i]
	}
	return token[:0]
}
