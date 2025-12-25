package storage

import (
	"context"
	"sync"
	"time"
)

type tokenCacheEntry struct {
	userID    string
	expiresAt time.Time
}

type MemoryTokenCache struct {
	mu      sync.RWMutex
	entries map[string]tokenCacheEntry

	done     chan struct{}
	interval time.Duration
}

func NewMemoryTokenCache(cleanupInterval time.Duration) *MemoryTokenCache {
	c := &MemoryTokenCache{
		entries:  make(map[string]tokenCacheEntry),
		done:     make(chan struct{}),
		interval: cleanupInterval,
	}
	go c.cleanupLoop()
	return c
}

func (c *MemoryTokenCache) GetUserID(_ context.Context, tokenHash string) (string, error) {
	c.mu.RLock()
	entry, ok := c.entries[tokenHash]
	c.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return "", ErrNotFound
	}
	return entry.userID, nil
}

func (c *MemoryTokenCache) SetUserID(_ context.Context, tokenHash string, userID string, ttl time.Duration) error {
	c.mu.Lock()
	c.entries[tokenHash] = tokenCacheEntry{
		userID:    userID,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
	return nil
}

func (c *MemoryTokenCache) cleanupLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.done:
			return
		}
	}
}

func (c *MemoryTokenCache) cleanup() {
	now := time.Now()
	c.mu.Lock()
	for hash, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, hash)
		}
	}
	c.mu.Unlock()
}

func (c *MemoryTokenCache) Close() error {
	close(c.done)
	return nil
}
