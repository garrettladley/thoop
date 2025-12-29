package xcontext

import "context"

type shutdownInProgressKey struct{}

// SetShutdownInProgress marks the context as being in a shutdown state.
// This allows handlers to distinguish between normal client disconnects
// and server-initiated shutdowns for better logging and cleanup.
func SetShutdownInProgress(ctx context.Context, inProgress bool) context.Context {
	return context.WithValue(ctx, shutdownInProgressKey{}, inProgress)
}

// IsShutdownInProgress checks if the context is marked as being in a shutdown state.
// Returns false if the shutdown flag is not set or is explicitly set to false.
func IsShutdownInProgress(ctx context.Context) bool {
	inProgress, ok := ctx.Value(shutdownInProgressKey{}).(bool)
	return ok && inProgress
}
