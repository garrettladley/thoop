package storage

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var _ Backend = (*MemoryBackend)(nil)

type stateWithTTL struct {
	entry StateEntry
	ttl   time.Duration
}

type MemoryBackend struct {
	// Rate limiting
	limiters  map[string]*rate.Limiter
	limiterMu sync.RWMutex
	rateLimit rate.Limit
	rateBurst int

	// State storage
	states   map[string]stateWithTTL
	statesMu sync.RWMutex

	// Cleanup
	done chan struct{}
}

func NewMemoryBackend(ratePerSec float64, burst int) *MemoryBackend {
	m := &MemoryBackend{
		limiters:  make(map[string]*rate.Limiter),
		rateLimit: rate.Limit(ratePerSec),
		rateBurst: burst,
		states:    make(map[string]stateWithTTL),
		done:      make(chan struct{}),
	}

	go m.cleanupLoop()

	return m
}

func (m *MemoryBackend) Allow(_ context.Context, key string) (bool, error) {
	m.limiterMu.RLock()
	limiter, exists := m.limiters[key]
	m.limiterMu.RUnlock()

	if exists {
		return limiter.Allow(), nil
	}

	m.limiterMu.Lock()
	defer m.limiterMu.Unlock()

	limiter, exists = m.limiters[key]
	if exists {
		return limiter.Allow(), nil
	}

	limiter = rate.NewLimiter(m.rateLimit, m.rateBurst)
	m.limiters[key] = limiter
	return limiter.Allow(), nil
}

func (m *MemoryBackend) Set(_ context.Context, state string, entry StateEntry, ttl time.Duration) error {
	m.statesMu.Lock()
	m.states[state] = stateWithTTL{entry: entry, ttl: ttl}
	m.statesMu.Unlock()
	return nil
}

func (m *MemoryBackend) GetAndDelete(_ context.Context, state string) (StateEntry, error) {
	m.statesMu.Lock()
	s, ok := m.states[state]
	if ok {
		delete(m.states, state)
	}
	m.statesMu.Unlock()

	if !ok {
		return StateEntry{}, ErrNotFound
	}

	if time.Since(s.entry.CreatedAt) > s.ttl {
		return StateEntry{}, ErrNotFound
	}

	return s.entry, nil
}

func (m *MemoryBackend) Close() error {
	close(m.done)
	return nil
}

func (m *MemoryBackend) Ping(_ context.Context) error {
	return nil
}

func (m *MemoryBackend) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.statesMu.Lock()
			now := time.Now()
			for state, s := range m.states {
				if now.Sub(s.entry.CreatedAt) > s.ttl {
					delete(m.states, state)
				}
			}
			m.statesMu.Unlock()
		case <-m.done:
			return
		}
	}
}
