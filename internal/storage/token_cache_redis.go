package storage

import (
	"context"
	"errors"
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

func (c *RedisTokenCache) GetUserID(ctx context.Context, tokenHash string) (string, error) {
	userID, err := c.client.Get(ctx, tokenCacheKeyPrefix+tokenHash).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (c *RedisTokenCache) SetUserID(ctx context.Context, tokenHash string, userID string, ttl time.Duration) error {
	return c.client.Set(ctx, tokenCacheKeyPrefix+tokenHash, userID, ttl).Err()
}
