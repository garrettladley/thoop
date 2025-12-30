package storage

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed ratelimit.lua
var rateLimitLua string

var rateLimitScript = redis.NewScript(rateLimitLua)

type rateLimitParams struct {
	window time.Duration // ARGV[1]: sliding window size in milliseconds
	limit  int           // ARGV[2]: max requests allowed in window
	ttl    time.Duration // ARGV[3]: key expiration in seconds
}

func (p rateLimitParams) args() []any {
	return []any{
		p.window.Milliseconds(),
		p.limit,
		int(p.ttl.Seconds()),
	}
}

func runRateLimitScript(ctx context.Context, client *redis.Client, key string, params rateLimitParams) (bool, error) {
	result, err := rateLimitScript.Run(ctx, client,
		[]string{key},
		params.args()...,
	).Int()
	if err != nil {
		return false, fmt.Errorf("failed to run rate limit script: %w", err)
	}
	return result == 1, nil
}
