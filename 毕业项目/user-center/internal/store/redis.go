package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr, password string, db int) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) LoginFailures(ctx context.Context, email, ip string) (int, error) {
	count, err := s.client.Get(ctx, loginFailKey(email, ip)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

func (s *RedisStore) RecordLoginFailure(ctx context.Context, email, ip string, window time.Duration) error {
	key := loginFailKey(email, ip)
	pipe := s.client.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisStore) ClearLoginFailures(ctx context.Context, email, ip string) error {
	return s.client.Del(ctx, loginFailKey(email, ip)).Err()
}

func (s *RedisStore) RevokeAccessToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return s.client.Set(ctx, accessRevokeKey(tokenID), "1", ttl).Err()
}

func (s *RedisStore) IsAccessTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	exists, err := s.client.Exists(ctx, accessRevokeKey(tokenID)).Result()
	return exists > 0, err
}

func loginFailKey(email, ip string) string {
	return fmt.Sprintf("uc:login_fail:%s:%s", email, ip)
}

func accessRevokeKey(tokenID string) string {
	return "uc:access_revoked:" + tokenID
}
