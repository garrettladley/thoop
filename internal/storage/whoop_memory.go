package storage

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// userLimiters holds the rate limiters for a single user.
type userLimiters struct {
	minuteLimiter *rate.Limiter
	dayLimiter    *rate.Limiter
	lastAccess    time.Time
}

// WhoopMemoryLimiter implements WhoopRateLimiter using in-memory rate limiters.
type WhoopMemoryLimiter struct {
	config WhoopRateLimiterConfig

	// Per-user limiters
	userLimiters map[string]*userLimiters
	userMu       sync.RWMutex

	// Global limiters
	globalMinuteLimiter *rate.Limiter
	globalDayLimiter    *rate.Limiter
	globalMu            sync.RWMutex

	// Cleanup
	done            chan struct{}
	cleanupInterval time.Duration
}

// NewWhoopMemoryLimiter creates a new in-memory WHOOP rate limiter.
func NewWhoopMemoryLimiter(config WhoopRateLimiterConfig) *WhoopMemoryLimiter {
	w := &WhoopMemoryLimiter{
		config:              config,
		userLimiters:        make(map[string]*userLimiters),
		globalMinuteLimiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(config.GlobalMinuteLimit)), config.GlobalMinuteLimit),
		globalDayLimiter:    rate.NewLimiter(rate.Every(24*time.Hour/time.Duration(config.GlobalDayLimit)), config.GlobalDayLimit),
		done:                make(chan struct{}),
		cleanupInterval:     5 * time.Minute,
	}
	go w.cleanupLoop()
	return w
}

func (w *WhoopMemoryLimiter) getOrCreateUserLimiters(userKey string) *userLimiters {
	now := time.Now()

	w.userMu.RLock()
	limiters, exists := w.userLimiters[userKey]
	w.userMu.RUnlock()

	if exists {
		// update lastAccess under write lock
		w.userMu.Lock()
		limiters.lastAccess = now
		w.userMu.Unlock()
		return limiters
	}

	w.userMu.Lock()
	defer w.userMu.Unlock()

	limiters, exists = w.userLimiters[userKey]
	if exists {
		limiters.lastAccess = now
		return limiters
	}

	limiters = &userLimiters{
		minuteLimiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(w.config.PerUserMinuteLimit)), w.config.PerUserMinuteLimit),
		dayLimiter:    rate.NewLimiter(rate.Every(24*time.Hour/time.Duration(w.config.PerUserDayLimit)), w.config.PerUserDayLimit),
		lastAccess:    now,
	}
	w.userLimiters[userKey] = limiters

	return limiters
}

func (w *WhoopMemoryLimiter) cleanupLoop() {
	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.cleanup()
		case <-w.done:
			return
		}
	}
}

func (w *WhoopMemoryLimiter) cleanup() {
	threshold := time.Now().Add(-25 * time.Hour)

	w.userMu.Lock()
	for key, limiters := range w.userLimiters {
		if limiters.lastAccess.Before(threshold) {
			delete(w.userLimiters, key)
		}
	}
	w.userMu.Unlock()
}

func (w *WhoopMemoryLimiter) Close() error {
	close(w.done)
	return nil
}

func (w *WhoopMemoryLimiter) CheckAndIncrement(ctx context.Context, userKey string) (*WhoopRateLimitState, error) {
	userLims := w.getOrCreateUserLimiters(userKey)

	now := time.Now()

	if !userLims.minuteLimiter.AllowN(now, 1) {
		reason := WhoopRateLimitReasonPerUserMinute
		return &WhoopRateLimitState{
			Allowed:     false,
			Reason:      &reason,
			MinuteReset: now.Add(time.Minute).Truncate(time.Minute),
		}, nil
	}

	if !userLims.dayLimiter.AllowN(now, 1) {
		reason := WhoopRateLimitReasonPerUserDay
		userLims.minuteLimiter.AllowN(now.Add(-time.Nanosecond), -1)
		return &WhoopRateLimitState{
			Allowed:  false,
			Reason:   &reason,
			DayReset: now.Add(24 * time.Hour).Truncate(24 * time.Hour),
		}, nil
	}

	w.globalMu.Lock()
	globalMinAllowed := w.globalMinuteLimiter.AllowN(now, 1)
	if !globalMinAllowed {
		w.globalMu.Unlock()
		reason := WhoopRateLimitReasonGlobalMinute
		userLims.minuteLimiter.AllowN(now.Add(-time.Nanosecond), -1)
		userLims.dayLimiter.AllowN(now.Add(-time.Nanosecond), -1)
		return &WhoopRateLimitState{
			Allowed:     false,
			Reason:      &reason,
			MinuteReset: now.Add(time.Minute).Truncate(time.Minute),
		}, nil
	}

	globalDayAllowed := w.globalDayLimiter.AllowN(now, 1)
	w.globalMu.Unlock()

	if !globalDayAllowed {
		userLims.minuteLimiter.AllowN(now.Add(-time.Nanosecond), -1)
		userLims.dayLimiter.AllowN(now.Add(-time.Nanosecond), -1)
		w.globalMu.Lock()
		w.globalMinuteLimiter.AllowN(now.Add(-time.Nanosecond), -1)
		w.globalMu.Unlock()
		reason := WhoopRateLimitReasonGlobalDay
		return &WhoopRateLimitState{
			Allowed:  false,
			Reason:   &reason,
			DayReset: now.Add(24 * time.Hour).Truncate(24 * time.Hour),
		}, nil
	}

	w.globalMu.RLock()
	minRemaining := w.globalMinuteLimiter.Tokens()
	dayRemaining := w.globalDayLimiter.Tokens()
	w.globalMu.RUnlock()

	return &WhoopRateLimitState{
		Allowed:         true,
		MinuteRemaining: int(minRemaining),
		DayRemaining:    int(dayRemaining),
		MinuteReset:     now.Add(time.Minute).Truncate(time.Minute),
		DayReset:        now.Add(24 * time.Hour).Truncate(24 * time.Hour),
	}, nil
}

func (w *WhoopMemoryLimiter) UpdateFromHeaders(ctx context.Context, headers http.Header) error {
	limitStr := headers.Get("X-RateLimit-Limit")
	remainingStr := headers.Get("X-RateLimit-Remaining")

	if limitStr == "" || remainingStr == "" {
		return nil
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return fmt.Errorf("failed to parse X-RateLimit-Limit: %w", err)
	}

	remaining, err := strconv.Atoi(remainingStr)
	if err != nil {
		return fmt.Errorf("failed to parse X-RateLimit-Remaining: %w", err)
	}

	w.globalMu.Lock()
	defer w.globalMu.Unlock()

	now := time.Now()

	switch limit {
	case 100:
		tokensToAdjust := remaining - int(w.globalMinuteLimiter.Tokens())
		if tokensToAdjust != 0 {
			w.globalMinuteLimiter.AllowN(now, -tokensToAdjust)
		}
	case 10000:
		tokensToAdjust := remaining - int(w.globalDayLimiter.Tokens())
		if tokensToAdjust != 0 {
			w.globalDayLimiter.AllowN(now, -tokensToAdjust)
		}
	}

	return nil
}

func (w *WhoopMemoryLimiter) GetUserStats(ctx context.Context, userKey string) (*UserRateLimitStats, error) {
	w.userMu.RLock()
	limiters, exists := w.userLimiters[userKey]
	w.userMu.RUnlock()

	if !exists {
		return &UserRateLimitStats{
			MinuteCount: 0,
			DayCount:    0,
		}, nil
	}

	minUsed := w.config.PerUserMinuteLimit - int(limiters.minuteLimiter.Tokens())
	dayUsed := w.config.PerUserDayLimit - int(limiters.dayLimiter.Tokens())

	if minUsed < 0 {
		minUsed = 0
	}
	if dayUsed < 0 {
		dayUsed = 0
	}

	return &UserRateLimitStats{
		MinuteCount: minUsed,
		DayCount:    dayUsed,
	}, nil
}

func (w *WhoopMemoryLimiter) GetGlobalStats(ctx context.Context) (*GlobalRateLimitStats, error) {
	w.globalMu.RLock()
	defer w.globalMu.RUnlock()

	minRemaining := int(w.globalMinuteLimiter.Tokens())
	dayRemaining := int(w.globalDayLimiter.Tokens())

	if minRemaining < 0 {
		minRemaining = 0
	}
	if dayRemaining < 0 {
		dayRemaining = 0
	}

	return &GlobalRateLimitStats{
		MinuteRemaining: minRemaining,
		DayRemaining:    dayRemaining,
	}, nil
}
