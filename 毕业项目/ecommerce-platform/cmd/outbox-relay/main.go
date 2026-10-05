package main

import (
	"context"
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
		kgo.DefaultProduceTopic(cfg.KafkaTopic),
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

	log.Printf("outbox relay publishing to %s", cfg.KafkaTopic)
	for ctx.Err() == nil {
		events, err := db.FetchOutbox(ctx, 20)
		if err != nil {
			log.Printf("fetch outbox: %v", err)
			sleep(ctx, cfg.PollInterval)
			continue
		}
		if len(events) == 0 {
			sleep(ctx, cfg.PollInterval)
			continue
		}
		for _, event := range events {
			record := &kgo.Record{
				Key:   []byte(event.AggregateID),
				Value: event.Payload,
				Headers: []kgo.RecordHeader{
					{Key: "event_id", Value: []byte(event.EventID)},
					{Key: "event_type", Value: []byte(event.EventType)},
				},
			}
			result := client.ProduceSync(ctx, record)
			if err := result.FirstErr(); err != nil {
				log.Printf("publish event %s failed: %v", event.EventID, err)
				_ = db.MarkOutboxRetry(ctx, event.ID)
				continue
			}
			if err := db.MarkOutboxSent(ctx, event.ID); err != nil {
				log.Printf("mark event %s sent failed: %v", event.EventID, err)
			}
		}
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
