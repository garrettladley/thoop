package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/xcontext"
)

// ShutdownContext is middleware that detects when the base context has been cancelled
// (indicating server shutdown) and marks the request context accordingly.
//
// This allows handlers to distinguish between client disconnects and server-initiated
// shutdowns, enabling different logging and cleanup behavior.
func ShutdownContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// If the base context is cancelled, mark shutdown in progress
		if ctx.Err() != nil {
			ctx = xcontext.SetShutdownInProgress(ctx, true)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
