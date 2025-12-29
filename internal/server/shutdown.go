package server

import (
	"context"
	"time"
)

// ShutdownCoordinator manages graceful shutdown of the HTTP server,
// particularly for long-lived connections like SSE.
type ShutdownCoordinator struct {
	baseCtx     context.Context
	cancel      context.CancelFunc
	gracePeriod time.Duration
}

// NewShutdownCoordinator creates a new shutdown coordinator with the specified grace period.
// The grace period determines how long to wait for active connections (particularly SSE)
// to close gracefully before calling server.Shutdown().
func NewShutdownCoordinator(gracePeriod time.Duration) *ShutdownCoordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &ShutdownCoordinator{
		baseCtx:     ctx,
		cancel:      cancel,
		gracePeriod: gracePeriod,
	}
}

// BaseContext returns the base context for all HTTP requests.
// This context is cancelled when shutdown is initiated, allowing
// long-lived connections to detect shutdown and close gracefully.
func (sc *ShutdownCoordinator) BaseContext() context.Context {
	return sc.baseCtx
}

// InitiateShutdown cancels the base context and waits for the grace period.
// This gives active connections (particularly SSE) time to send final messages
// and close cleanly before server.Shutdown() is called.
//
// This function blocks for the duration of the grace period.
func (sc *ShutdownCoordinator) InitiateShutdown() {
	// Cancel the base context, which signals all active connections
	// that shutdown is beginning
	sc.cancel()

	// Wait for the grace period to allow connections to close gracefully
	time.Sleep(sc.gracePeriod)
}
