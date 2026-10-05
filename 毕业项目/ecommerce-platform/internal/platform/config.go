package platform

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr        string
	InventoryAddr   string
	InventoryTarget string
	MySQLDSN        string
	RedisAddr       string
	KafkaBrokers    []string
	KafkaTopic      string
	KafkaGroup      string
	PollInterval    time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:        getenv("EC_HTTP_ADDR", ":8081"),
		InventoryAddr:   getenv("EC_INVENTORY_ADDR", ":9091"),
		InventoryTarget: getenv("EC_INVENTORY_TARGET", "inventory:9091"),
		MySQLDSN:        getenv("EC_MYSQL_DSN", "ecommerce:ecommerce-local@tcp(127.0.0.1:3307)/ecommerce?parseTime=true&loc=Local"),
		RedisAddr:       getenv("EC_REDIS_ADDR", "127.0.0.1:6380"),
		KafkaBrokers:    split(getenv("EC_KAFKA_BROKERS", "127.0.0.1:9092")),
		KafkaTopic:      getenv("EC_KAFKA_TOPIC", "order-events"),
		KafkaGroup:      getenv("EC_KAFKA_GROUP", "payment-worker"),
		PollInterval:    time.Duration(getenvInt("EC_POLL_INTERVAL_MS", 1000)) * time.Millisecond,
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func split(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
