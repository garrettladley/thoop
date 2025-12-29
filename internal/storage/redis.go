package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	go_json "github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

var _ Backend = (*RedisBackend)(nil)

const (
	rateLimitKeyPrefix = "ratelimit:"
	stateKeyPrefix     = "state:"
)

type RedisConfig struct {
	Client *redis.Client
}

type RedisBackend struct {
	client     *redis.Client
	rateLimit  int
	rateWindow time.Duration
}

func NewRedisBackend(cfg RedisConfig, rateLimit int) (*RedisBackend, error) {
	return &RedisBackend{
		client:     cfg.Client,
		rateLimit:  rateLimit,
		rateWindow: time.Second,
	}, nil
}

func (r *RedisBackend) Allow(ctx context.Context, key string) (RateLimitResult, error) {
	params := rateLimitParams{
		window: r.rateWindow,
		limit:  r.rateLimit,
		ttl:    r.rateWindow + time.Second,
	}

	allowed, err := runRateLimitScript(ctx, r.client, rateLimitKeyPrefix+key, params)
	if err != nil {
		return RateLimitResult{}, fmt.Errorf("failed to run rate limit script: %w", err)
	}

	return RateLimitResult{
		Allowed:    allowed,
		RetryAfter: r.rateWindow,
	}, nil
}

func (r *RedisBackend) Set(ctx context.Context, state string, entry StateEntry, ttl time.Duration) error {
	data, err := go_json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal state entry: %w", err)
	}

	if err := r.client.Set(ctx, stateKeyPrefix+state, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set state: %w", err)
	}

	return nil
}

func (r *RedisBackend) GetAndDelete(ctx context.Context, state string) (StateEntry, error) {
	key := stateKeyPrefix + state

	data, err := r.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return StateEntry{}, ErrNotFound
	}
	if err != nil {
		return StateEntry{}, fmt.Errorf("failed to get and delete state: %w", err)
	}

	var entry StateEntry
	if err := go_json.Unmarshal(data, &entry); err != nil {
		return StateEntry{}, fmt.Errorf("failed to unmarshal state entry: %w", err)
	}

	return entry, nil
}

func (r *RedisBackend) Close() error {
	return r.client.Close()
}

func (r *RedisBackend) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
