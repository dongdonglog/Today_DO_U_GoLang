package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-book/graduate/ecommerce-platform/internal/platform"
	"github.com/go-book/graduate/ecommerce-platform/internal/store"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	cfg := platform.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers...),
		kgo.ConsumerGroup(cfg.KafkaGroup),
		kgo.ConsumeTopics(cfg.KafkaTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		log.Fatalf("connect kafka: %v", err)
	}
	defer client.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		cancel()
	}()

	log.Printf("payment worker consuming %s as %s", cfg.KafkaTopic, cfg.KafkaGroup)
	for ctx.Err() == nil {
		pollCtx, pollCancel := context.WithTimeout(ctx, time.Second)
		fetches := client.PollRecords(pollCtx, 50)
		pollCancel()
		if fetches.IsClientClosed() {
			return
		}
		fetches.EachRecord(func(record *kgo.Record) {
			var event store.OrderEvent
			if err := json.Unmarshal(record.Value, &event); err != nil {
				log.Printf("skip broken event: %v", err)
				client.CommitRecords(ctx, record)
				return
			}
			if event.EventType != "order.created" {
				client.CommitRecords(ctx, record)
				return
			}
			if err := db.MarkOrderPaid(ctx, event, "payment-worker"); err != nil {
				log.Printf("mark order paid failed, will retry: %v", err)
				return
			}
			client.CommitRecords(ctx, record)
		})
	}
}
