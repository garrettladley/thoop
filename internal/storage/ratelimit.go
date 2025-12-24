package storage

import (
	"context"
	_ "embed"

	"github.com/redis/go-redis/v9"
)

//go:embed ratelimit.lua
var rateLimitLua string

var rateLimitScript = redis.NewScript(rateLimitLua)

type rateLimitParams struct {
	windowMs   int64 // ARGV[1]: sliding window size in milliseconds
	limit      int   // ARGV[2]: max requests allowed in window
	ttlSeconds int   // ARGV[3]: key expiration in seconds
}

func (p rateLimitParams) args() []any {
	return []any{p.windowMs, p.limit, p.ttlSeconds}
}

func runRateLimitScript(ctx context.Context, client *redis.Client, key string, params rateLimitParams) (bool, error) {
	result, err := rateLimitScript.Run(ctx, client,
		[]string{key},
		params.args()...,
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
