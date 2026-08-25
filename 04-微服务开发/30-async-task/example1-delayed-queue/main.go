package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// 延迟任务:订单创建后 15 分钟未支付则自动关闭。
//
//	go run . -role=produce   # 创建 3 笔订单(1 笔 2 秒后到期,2 笔 30 秒)
//	go run . -role=consume   # 轮询到期的订单并"关闭"(演示中只打印)
//
// 为什么不用 time.AfterFunc / timer 轮询数据库?
//   - 进程重启定时器全丢;ZSET 在 Redis 里持久化,多实例共享
//   - 数据库轮询对大表是灾难;ZSET 按分数(到期时间)索引,O(logN) 取最小
//
// 核心数据结构:
//
//	ZADD delay_queue <到期时间戳> <任务ID>
type orderTask struct {
	OrderNo   string `json:"order_no"`
	Action    string `json:"action"` // close_order
	CreatedAt int64  `json:"created_at"`
}

const (
	queueKey = "delay:order-close"
	bucket   = 5 // 扫描提前量(秒):取 [0, now+bucket] 的任务,避免边界漏单
)

func main() {
	role := "consume"
	for _, a := range os.Args[1:] {
		if len(a) > 6 && a[:6] == "-role=" {
			role = a[6:]
		}
	}

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()
	ctx := context.Background()

	if role == "produce" {
		produce(ctx, rdb)
		return
	}
	consume(ctx, rdb)
}

func produce(ctx context.Context, rdb *redis.Client) {
	now := time.Now()
	tasks := []struct {
		no    string
		delay time.Duration
	}{
		{"NO-DQ-0001", 2 * time.Second}, // 很快到期,便于观察
		{"NO-DQ-0002", 30 * time.Second},
		{"NO-DQ-0003", 30 * time.Second},
	}
	for _, t := range tasks {
		payload, _ := json.Marshal(orderTask{OrderNo: t.no, Action: "close_order", CreatedAt: now.Unix()})
		score := float64(now.Add(t.delay).Unix())
		if err := rdb.ZAdd(ctx, queueKey, redis.Z{Score: score, Member: string(payload)}).Err(); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("订单 %s 将在 %s 后检查(close)\n", t.no, t.delay)
	}
}

func consume(ctx context.Context, rdb *redis.Client) {
	fmt.Println("扫描延迟任务,Ctrl+C 退出")
	for {
		now := float64(time.Now().Unix())

		// 1. 取出到期任务(score ∈ [0, now+bucket])
		msgs, err := rdb.ZRangeByScore(ctx, queueKey, &redis.ZRangeBy{
			Min: "-inf", Max: fmt.Sprintf("%f", now+bucket),
		}).Result()
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}

		for _, m := range msgs {
			var task orderTask
			if json.Unmarshal([]byte(m), &task) != nil {
				continue // 坏消息:记日志后从 ZSET 移除,别让它永远留在队里
			}

			// 2. 抢占:ZREM 成功=本实例获得执行权(ZSET 的原子删除当锁用)。
			//    多实例同时扫到同一任务时,只有一个 ZREM 返回 1,天然防重复执行。
			removed, err := rdb.ZRem(ctx, queueKey, m).Result()
			if err != nil || removed == 0 {
				continue // 已被其他实例抢走
			}

			// 3. 执行业务。真实场景:执行失败要把任务塞回队列(降分重试)或进死信
			fmt.Printf("[%s] 关闭超时订单 %s\n",
				time.Now().Format("15:04:05"), task.OrderNo)
		}

		time.Sleep(500 * time.Millisecond)
	}
}
