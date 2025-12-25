package storage

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/redis/go-redis/v9"
)

//go:embed whoop_ratelimit.lua
var whoopRateLimitLua string

var whoopRateLimitScript = redis.NewScript(whoopRateLimitLua)

const (
	whoopUserKeyPrefix   = "whoop:ratelimit:user:"
	whoopGlobalKeyPrefix = "whoop:ratelimit:global"
)

type WhoopRateLimiterConfig struct {
	PerUserMinuteLimit int // default: 20
	PerUserDayLimit    int // default: 2000
	GlobalMinuteLimit  int // default: 95
	GlobalDayLimit     int // default: 9950
}

type WhoopRedisLimiter struct {
	client *redis.Client
	config WhoopRateLimiterConfig
}

func NewWhoopRedisLimiter(cfg RedisConfig, config WhoopRateLimiterConfig) *WhoopRedisLimiter {
	return &WhoopRedisLimiter{
		client: cfg.Client,
		config: config,
	}
}

func (w *WhoopRedisLimiter) CheckAndIncrement(ctx context.Context, userKey string) (*WhoopRateLimitState, error) {
	keys := []string{
		whoopUserKeyPrefix + userKey + ":minute",
		whoopUserKeyPrefix + userKey + ":day",
		whoopGlobalKeyPrefix + ":minute",
		whoopGlobalKeyPrefix + ":day",
	}

	const (
		minuteWindowMs = 60_000     // minute window in ms
		dayWindowMs    = 86_400_000 // day window in ms
		ttlSeconds     = 90_000     // TTL in seconds (25 hours for safety)
	)
	args := []any{
		w.config.PerUserMinuteLimit,
		w.config.PerUserDayLimit,
		w.config.GlobalMinuteLimit,
		w.config.GlobalDayLimit,
		minuteWindowMs,
		dayWindowMs,
		ttlSeconds,
	}

	result, err := whoopRateLimitScript.Run(ctx, w.client, keys, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to run WHOOP rate limit script: %w", err)
	}

	resultSlice, ok := result.([]any)
	if !ok || len(resultSlice) < 2 {
		return nil, fmt.Errorf("unexpected result format from rate limit script")
	}

	allowed, ok := resultSlice[0].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected allowed value type")
	}

	state := &WhoopRateLimitState{
		Allowed: allowed == 1,
	}

	if allowed == 1 {
		minRemaining, ok := resultSlice[1].(int64)
		if !ok {
			return nil, fmt.Errorf("unexpected minute remaining type")
		}
		dayRemaining, ok := resultSlice[2].(int64)
		if !ok {
			return nil, fmt.Errorf("unexpected day remaining type")
		}

		state.MinuteRemaining = int(minRemaining)
		state.DayRemaining = int(dayRemaining)
		state.MinuteReset = time.Now().Add(time.Minute).Truncate(time.Minute)
		state.DayReset = time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
	} else {
		reasonStr, ok := resultSlice[1].(string)
		if !ok {
			return nil, fmt.Errorf("unexpected reason type: got %T", resultSlice[1])
		}
		reason := WhoopRateLimitReason(reasonStr)
		state.Reason = &reason
		if reason == WhoopRateLimitReasonPerUserMinute || reason == WhoopRateLimitReasonGlobalMinute {
			state.MinuteReset = time.Now().Add(time.Minute).Truncate(time.Minute)
		} else {
			state.DayReset = time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
		}
	}

	return state, nil
}

func (w *WhoopRedisLimiter) UpdateFromHeaders(ctx context.Context, headers http.Header) error {
	info, err := whoop.ParseRateLimitHeaders(headers)
	if err != nil {
		return fmt.Errorf("failed to parse rate limit headers: %w", err)
	}
	if info == nil {
		return nil
	}

	// determine which window based on limit value
	// WHOOP returns either 100 (minute) or 10000 (day)
	var key string
	switch info.Limit {
	case 100:
		key = whoopGlobalKeyPrefix + ":minute"
	case 10_000:
		key = whoopGlobalKeyPrefix + ":day"
	default:
		return nil
	}

	used := info.Limit - info.Remaining

	timeResult, err := w.client.Time(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to get Redis time: %w", err)
	}
	now := timeResult.UnixMilli()

	// clear existing entries and set new ones based on WHOOP's count
	// this syncs our local state with WHOOP's actual state
	pipe := w.client.Pipeline()
	pipe.Del(ctx, key)

	// add 'used' number of entries with current timestamp
	// this is an approximation since we don't know exact timestamps
	for i := range used {
		member := fmt.Sprintf("%d:%d", now, i)
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: member})
	}

	// set expiration based on reset duration
	ttl := info.Reset + time.Minute // add margin
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update rate limit state from headers: %w", err)
	}

	return nil
}

func (w *WhoopRedisLimiter) GetUserStats(ctx context.Context, userKey string) (*UserRateLimitStats, error) {
	minKey := whoopUserKeyPrefix + userKey + ":minute"
	dayKey := whoopUserKeyPrefix + userKey + ":day"

	timeResult, err := w.client.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis time: %w", err)
	}
	now := timeResult.UnixMilli()

	minWindowStart := now - 60_000
	dayWindowStart := now - 86_400_000

	pipe := w.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, minKey, "-inf", strconv.FormatInt(minWindowStart, 10))
	pipe.ZRemRangeByScore(ctx, dayKey, "-inf", strconv.FormatInt(dayWindowStart, 10))
	minCountCmd := pipe.ZCard(ctx, minKey)
	dayCountCmd := pipe.ZCard(ctx, dayKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return &UserRateLimitStats{
		MinuteCount: int(minCountCmd.Val()),
		DayCount:    int(dayCountCmd.Val()),
	}, nil
}

func (w *WhoopRedisLimiter) GetGlobalStats(ctx context.Context) (*GlobalRateLimitStats, error) {
	minKey := whoopGlobalKeyPrefix + ":minute"
	dayKey := whoopGlobalKeyPrefix + ":day"

	timeResult, err := w.client.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis time: %w", err)
	}
	now := timeResult.UnixMilli()

	minWindowStart := now - 60_000
	dayWindowStart := now - 86_400_000

	pipe := w.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, minKey, "-inf", strconv.FormatInt(minWindowStart, 10))
	pipe.ZRemRangeByScore(ctx, dayKey, "-inf", strconv.FormatInt(dayWindowStart, 10))
	minCountCmd := pipe.ZCard(ctx, minKey)
	dayCountCmd := pipe.ZCard(ctx, dayKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get global stats: %w", err)
	}

	return &GlobalRateLimitStats{
		MinuteRemaining: w.config.GlobalMinuteLimit - int(minCountCmd.Val()),
		DayRemaining:    w.config.GlobalDayLimit - int(dayCountCmd.Val()),
	}, nil
}
