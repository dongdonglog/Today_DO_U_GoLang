package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// 消费组演示:同一个 group 里的消费者自动分摊分区。
//
//	终端 A: go run . -name=c1   # 启动消费者 1
//	终端 B: go run . -name=c2   # 启动消费者 2 → 触发再均衡,A/B 各分到部分分区
//	终端 C: go run . -role=produce
//
// 观察:
//   - B 加入时 A 日志打印 "分区被回收/重新分配"(cooperative 再均衡只动受影响的分区)
//   - 每条消息只被组内一个消费者处理(分区是分配的最小单位)
//   - kill 掉 B,它的分区自动移交回 A —— 这就是消费组的故障转移
func main() {
	name := "c1"
	role := "consume"
	for _, a := range os.Args[1:] {
		if len(a) > 6 && a[:6] == "-role=" {
			role = a[6:]
		}
		if len(a) > 6 && a[:6] == "-name=" {
			name = a[6:]
		}
	}

	const (
		brokers = "localhost:9092"
		topic   = "order-events"
		group   = "billing-service"
	)

	if role == "produce" {
		cl, err := kgo.NewClient(kgo.SeedBrokers(brokers), kgo.DefaultProduceTopic(topic))
		if err != nil {
			log.Fatal(err)
		}
		defer cl.Close()
		ctx := context.Background()
		for i := 11; i <= 20; i++ {
			cl.Produce(ctx, &kgo.Record{
				Key:   []byte(fmt.Sprintf("NO-2024-%04d", i)),
				Value: []byte(fmt.Sprintf(`{"order_no":"NO-2024-%04d","action":"paid"}`, i)),
			}, nil)
		}
		if err := cl.Flush(ctx); err != nil {
			log.Fatal(err)
		}
		fmt.Println("已发 10 条 paid 事件")
		return
	}

	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),
		kgo.Balancers(kgo.CooperativeStickyBalancer()), // 协作式:再均衡时只回收必要的分区
		kgo.OnPartitionsAssigned(func(_ context.Context, cl *kgo.Client, assigned map[string][]int32) {
			for t, ps := range assigned {
				fmt.Printf("[%s] 分配到 %s 分区 %v\n", name, t, ps)
			}
		}),
		kgo.OnPartitionsRevoked(func(_ context.Context, cl *kgo.Client, revoked map[string][]int32) {
			for t, ps := range revoked {
				fmt.Printf("[%s] 分区被回收 %s %v\n", name, t, ps)
			}
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cl.Close()

	ctx := context.Background()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	fmt.Printf("[%s] 开始消费,Ctrl+C 退出\n", name)

	for {
		pollCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		fetches := cl.PollRecords(pollCtx, 50) // 消息不足 50 条时等待超时到期返回,见 29.3
		cancel()
		if fetches.IsClientClosed() {
			return
		}
		fetches.EachRecord(func(r *kgo.Record) {
			fmt.Printf("[%s] [p%d o%d] %s\n",
				name, r.Partition, r.Offset, string(r.Value))
			cl.CommitRecords(ctx, r)
		})
	}
}
