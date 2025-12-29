package storage

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var ErrNotFound = errors.New("state not found")

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type StateEntry struct {
	LocalPort     string    `json:"local_port"`
	ClientVersion string    `json:"client_version"`
	CreatedAt     time.Time `json:"created_at"`
}

type StateStore interface {
	Set(ctx context.Context, state string, entry StateEntry, ttl time.Duration) error

	// GetAndDelete atomically retrieves and removes a state entry.
	// Returns ErrNotFound if the state does not exist or has expired.
	GetAndDelete(ctx context.Context, state string) (StateEntry, error)
}

type Backend interface {
	RateLimiter
	StateStore

	Close() error

	Ping(ctx context.Context) error
}

type WhoopRateLimitReason string

const (
	WhoopRateLimitReasonPerUserMinute WhoopRateLimitReason = "per-user-minute"
	WhoopRateLimitReasonPerUserDay    WhoopRateLimitReason = "per-user-day"
	WhoopRateLimitReasonGlobalMinute  WhoopRateLimitReason = "global-minute"
	WhoopRateLimitReasonGlobalDay     WhoopRateLimitReason = "global-day"
)

type WhoopRateLimitState struct {
	Allowed         bool
	MinuteRemaining int
	MinuteReset     time.Time
	DayRemaining    int
	DayReset        time.Time
	Reason          *WhoopRateLimitReason
}

type UserRateLimitStats struct {
	MinuteCount int
	DayCount    int
}

type GlobalRateLimitStats struct {
	MinuteRemaining int
	DayRemaining    int
}

type WhoopRateLimiter interface {
	// CheckAndIncrement checks both per-user and global limits, incrementing counters if allowed.
	// userKey: hash of access token or user ID extracted from token
	// Returns combined state - allowed only if BOTH per-user AND global limits pass
	CheckAndIncrement(ctx context.Context, userKey string) (*WhoopRateLimitState, error)

	CheckAndIncrementGlobalOnly(ctx context.Context) (*WhoopRateLimitState, error)

	// UpdateFromHeaders updates global rate limit state from WHOOP API response headers.
	// Syncs global counters with WHOOP's actual values.
	UpdateFromHeaders(ctx context.Context, headers http.Header) error

	GetUserStats(ctx context.Context, userKey string) (*UserRateLimitStats, error)

	GetGlobalStats(ctx context.Context) (*GlobalRateLimitStats, error)
}

// TokenCache caches validated bearer tokens to avoid repeated API calls.
// Maps token hash -> WHOOP user ID.
type TokenCache interface {
	// GetUserID returns the cached user ID for a token hash.
	// Returns ErrNotFound if not cached or expired.
	GetUserID(ctx context.Context, tokenHash string) (string, error)

	// SetUserID caches a user ID for a token hash with TTL.
	SetUserID(ctx context.Context, tokenHash string, userID string, ttl time.Duration) error
}
