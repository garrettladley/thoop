package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const tokenCacheKeyPrefix = "whoop:token:user:"

type RedisTokenCache struct {
	client *redis.Client
}

func NewRedisTokenCache(cfg RedisConfig) *RedisTokenCache {
	return &RedisTokenCache{client: cfg.Client}
}

func (c *RedisTokenCache) GetUserID(ctx context.Context, tokenHash string) (int64, error) {
	userIDStr, err := c.client.Get(ctx, tokenCacheKeyPrefix+tokenHash).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get user id from cache: %w", err)
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user id: %w", err)
	}
	return userID, nil
}

func (c *RedisTokenCache) SetUserID(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error {
	if err := c.client.Set(ctx, tokenCacheKeyPrefix+tokenHash, strconv.FormatInt(userID, 10), ttl).Err(); err != nil {
		return fmt.Errorf("failed to set user id in cache: %w", err)
	}
	return nil
}
