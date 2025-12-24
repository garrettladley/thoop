package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	json "github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

var _ Backend = (*RedisBackend)(nil)

const (
	rateLimitKeyPrefix = "ratelimit:"
	stateKeyPrefix     = "state:"
)

type RedisConfig struct {
	URL string `env:"URL"`
}

type RedisBackend struct {
	client     *redis.Client
	rateLimit  int
	rateWindow time.Duration
}

func NewRedisBackend(cfg RedisConfig, rateLimit int) (*RedisBackend, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisBackend{
		client:     client,
		rateLimit:  rateLimit,
		rateWindow: time.Second,
	}, nil
}

func (r *RedisBackend) Allow(ctx context.Context, key string) (bool, error) {
	params := rateLimitParams{
		window: r.rateWindow,
		limit:  r.rateLimit,
		ttl:    r.rateWindow + time.Second,
	}

	allowed, err := runRateLimitScript(ctx, r.client, rateLimitKeyPrefix+key, params)
	if err != nil {
		return false, fmt.Errorf("failed to run rate limit script: %w", err)
	}

	return allowed, nil
}

func (r *RedisBackend) Set(ctx context.Context, state string, entry StateEntry, ttl time.Duration) error {
	data, err := json.Marshal(entry)
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
	if err := json.Unmarshal(data, &entry); err != nil {
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
