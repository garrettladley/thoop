package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("state not found")

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type StateEntry struct {
	LocalPort string    `json:"local_port"`
	CreatedAt time.Time `json:"created_at"`
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
