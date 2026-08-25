package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

// 最基础的 生产者 + 消费者:订单事件写入 Kafka,消费端读取并处理。
//
//	go run . -role=topic    # 建 topic(3 分区)
//	go run . -role=producer # 发 10 条订单事件
//	go run . -role=consumer # 持续消费并打印
//
// 核心语义预览:
//   - 生产:RecordAppend 异步攒批发送;Flush 保证退出前都发出
//   - 消费:PollRecords 拉取,处理完 CommitRecords 提交位移 => 至少一次(at-least-once)
func main() {
	role := "consumer"
	for _, a := range os.Args[1:] {
		if len(a) > 6 && a[:6] == "-role=" {
			role = a[6:]
		}
	}

	const (
		brokers = "localhost:9092"
		topic   = "order-events"
		group   = "billing-service"
	)

	switch role {
	case "topic":
		createTopic(brokers, topic)
	case "producer":
		produce(brokers, topic)
	default:
		consume(brokers, topic, group)
	}
}

// createTopic 用 kadm(管理客户端)建 3 分区的 topic。
// 分区数决定同组消费者的最大并行度(见 example2)。
func createTopic(brokers, topic string) {
	cl, err := kgo.NewClient(kgo.SeedBrokers(brokers))
	if err != nil {
		log.Fatal(err)
	}
	defer cl.Close()
	admin := kadm.NewClient(cl)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := admin.CreateTopic(ctx, 3, 1, nil, topic)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || errors.Is(err, kerr.TopicAlreadyExists) {
			fmt.Printf("topic %s 已存在\n", topic)
			return
		}
		log.Fatalf("建 topic 失败: %v", err)
	}
	fmt.Printf("topic %s 创建成功(3 分区)\n", resp.Topic)
}

type orderEvent struct {
	OrderNo string `json:"order_no"`
	UserID  int64  `json:"user_id"`
	Amount  int64  `json:"amount"` // 分
	Action  string `json:"action"` // created / paid / shipped
}

func produce(brokers, topic string) {
	// 生产者配置要点(全部使用 franz-go 的默认值或显式安全值):
	//   - acks=all(默认):等所有 ISR 副本落盘才确认,不丢消息
	//   - 幂等生产者默认开启:重试不会产生重复消息(同一分区上 exactly-once 写)
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers),
		kgo.DefaultProduceTopic(topic), // 省去每条记录指定 topic
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cl.Close()
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		ev := fmt.Sprintf(`{"order_no":"NO-2024-%04d","user_id":%d,"amount":%d,"action":"created"}`,
			i, 1000+i, i*1000)

		// 按 order_no 做 key:同一订单的事件永远进同一分区,保证分区内有序
		key := fmt.Sprintf("NO-2024-%04d", i)

		cl.Produce(ctx, &kgo.Record{Key: []byte(key), Value: []byte(ev)},
			func(r *kgo.Record, err error) {
				if err != nil {
					log.Printf("发送失败 %s: %v", key, err) // 生产环境:记日志+重试/告警
					return
				}
				fmt.Printf("已发送 -> partition=%d offset=%d key=%s\n",
					r.Partition, r.Offset, key)
			})
	}

	// Produce 是异步攒批的;退出前 Flush 确认全部落盘
	if err := cl.Flush(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("全部发送完成")
}

func consume(brokers, topic, group string) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),                        // 手动提交:处理完成才提交,保证 at-least-once
		kgo.Balancers(kgo.CooperativeStickyBalancer()), // 协作式再均衡,停机窗口小
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cl.Close()
	ctx := context.Background()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("消费中,Ctrl+C 退出")

	for {
		// PollRecords:拉取一批记录(最多 100 条),内部已完成组内分区分配。
		// 注意:消息少于 100 条时会阻塞等待直到 ctx 到期,所以给 500ms 超时的 ctx,
		// 保证演示消息少时也能周期性返回、响应 Ctrl+C(生产同理,避免永久阻塞)
		pollCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		fetches := cl.PollRecords(pollCtx, 100)
		cancel()
		if fetches.IsClientClosed() {
			return
		}
		fetches.EachError(func(t string, p int32, err error) {
			log.Printf("拉取出错 topic=%s partition=%d: %v", t, p, err)
		})

		var processed []*kgo.Record
		fetches.EachRecord(func(r *kgo.Record) {
			fmt.Printf("[p%d o%d] key=%s value=%s\n",
				r.Partition, r.Offset, string(r.Key), string(r.Value))
			processed = append(processed, r)
		})

		// 全部处理成功后手动提交位移。
		// 如果处理到一半崩溃:未提交的位移会被重新消费 => 可能重复,不会丢失
		if len(processed) > 0 {
			cl.CommitRecords(ctx, processed...)
		}
	}
}
