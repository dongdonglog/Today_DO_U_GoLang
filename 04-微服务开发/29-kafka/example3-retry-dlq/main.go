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

	"github.com/twmb/franz-go/pkg/kgo"
)

// 消费失败的处理范式:立即重试 → 有限退避重试 → 进死信队列(dead letter queue)。
//
//	go run . -role=produce-poison   # 发 3 条正常消息 + 1 条"毒消息"(永远处理失败)
//	go run . -role=consume          # 消费:正常消息成功;毒消息重试 3 次后进 DLQ
//	                                # 注意:毒消息不会阻塞后续消息的处理
//	kcat 查看 DLQ:
//	  docker exec go-book-kafka /opt/kafka/bin/kafka-console-consumer.sh \
//	    --bootstrap-server localhost:9092 --topic order-events-dlq --from-beginning
//
// 重复运行提示:消费组 retry-demo-group 的位移已提交,再次 -role=consume 不会重放旧消息
// (Kafka 消费组默认从上次提交的位置继续)。想重新演示,先重置位移:
//
//	docker exec go-book-kafka /opt/kafka/bin/kafka-consumer-groups.sh \
//	  --bootstrap-server localhost:9092 --group retry-demo-group \
//	  --reset-offsets --to-earliest --execute --topic order-events
//
// 为什么不能无限原地重试:一条坏消息会卡住整个分区(后续消息全部延迟)。
// 快速失败 + 落 DLQ + 人工/自动修复,才是生产做法。
func main() {
	role := "consume"
	for _, a := range os.Args[1:] {
		if len(a) > 6 && a[:6] == "-role=" {
			role = a[6:]
		}
	}

	const (
		brokers  = "localhost:9092"
		topic    = "order-events"     // 复用 example1 的主 topic
		dlqTopic = "order-events-dlq" // 死信队列
		group    = "retry-demo-group"
		maxRetry = 3 // 业务级重试次数(不含首次)
	)

	switch role {
	case "produce-poison":
		producePoison(brokers, topic)
	default:
		consumeWithRetry(brokers, topic, dlqTopic, group, maxRetry)
	}
}

func producePoison(brokers, topic string) {
	cl, err := kgo.NewClient(kgo.SeedBrokers(brokers), kgo.DefaultProduceTopic(topic))
	if err != nil {
		log.Fatal(err)
	}
	defer cl.Close()
	ctx := context.Background()

	msgs := []string{
		`{"order_no":"OK-0001","action":"created"}`,
		`{broken json!!!`, // 毒消息:反序列化必败
		`{"order_no":"OK-0002","action":"paid"}`,
		`{"order_no":"OK-0003","action":"shipped"}`,
	}
	for i, m := range msgs {
		cl.Produce(ctx, &kgo.Record{
			Key:   []byte(fmt.Sprintf("k%d", i)),
			Value: []byte(m),
			// 把原始元数据写进 header,DLQ 消费者可以追溯来源
			Headers: []kgo.RecordHeader{{Key: "source", Value: []byte(topic)}},
		}, nil)
	}
	if err := cl.Flush(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("已发 4 条(含 1 条毒消息)")
}

type handler func(ctx context.Context, value []byte) error

// handle 是业务处理函数:解析 JSON 并"处理订单"。
// 毒消息会在这里持续报错。
func handle(_ context.Context, value []byte) error {
	v := string(value)
	if !strings.Contains(v, "{") || strings.Contains(v, "broken") {
		return fmt.Errorf("非法消息: %q", v)
	}
	// 真实业务:落库、调下游……这里用小概率失败模拟偶发错误
	if rand.Intn(100) < 10 {
		return fmt.Errorf("模拟瞬时错误")
	}
	fmt.Printf("  处理成功: %s\n", v)
	return nil
}

// consumeWithRetry 的核心逻辑:
//
//	首次处理失败 → 原地快速重试 2 次(间隔 50ms,对付瞬时抖动)
//	仍失败       → 记录到 DLQ(带 retry_count / error 头),提交位移,继续下一条
func consumeWithRetry(brokers, mainTopic, dlq string, group string, maxRetry int) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(mainTopic),
		kgo.DisableAutoCommit(),
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
		// PollRecords 在消息数 < maxPollRecords 时会一直等待直到 ctx 到期。
		// 演示场景消息很少,给个 500ms 超时的 ctx,保证能周期性返回并响应 Ctrl+C;
		// 生产里这个 ctx 通常是请求级超时,而不是永远阻塞。
		pollCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		fetches := cl.PollRecords(pollCtx, 20)
		cancel()
		if fetches.IsClientClosed() {
			return
		}
		fetches.EachRecord(func(r *kgo.Record) {
			var err error
			for attempt := 0; attempt <= maxRetry; attempt++ {
				if attempt > 0 {
					time.Sleep(50 * time.Millisecond) // 简单固定间隔;生产建议指数退避+抖动
				}
				if err = handle(ctx, r.Value); err == nil {
					break
				}
				log.Printf("处理失败(第%d次): %v", attempt+1, err)
			}

			if err != nil {
				sendToDLQ(ctx, cl, dlq, r, err)
			}
			cl.CommitRecords(ctx, r) // 无论成败都推进位移:毒消息不能卡住分区
		})
	}
}

// sendToDLQ 把失败消息原样转存到死信 topic,附上诊断头。
func sendToDLQ(ctx context.Context, cl *kgo.Client, dlq string, r *kgo.Record, cause error) {
	headers := append([]kgo.RecordHeader{
		{Key: "dlq-error", Value: []byte(cause.Error())},
		{Key: "dlq-time", Value: []byte(time.Now().Format(time.RFC3339))},
	}, r.Headers...)

	wait := cl.ProduceSync(ctx, &kgo.Record{
		Topic:     dlq,
		Key:       r.Key,
		Value:     r.Value,
		Headers:   headers,
		Timestamp: time.Now(),
	})
	if wait.FirstErr() != nil {
		log.Printf("进 DLQ 失败!这条消息可能丢: key=%s err=%v",
			string(r.Key), wait.FirstErr()) // 生产环境:本地落盘兜底 + 高优告警
		return
	}
	log.Printf("已入死信队列 %s: key=%s cause=%v", dlq, string(r.Key), cause)
}
