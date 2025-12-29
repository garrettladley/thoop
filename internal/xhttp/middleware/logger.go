package middleware

import (
	"log/slog"
	"net/http"

	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xslog"
)

// Must run AFTER ID context setting middlewares.
func Logger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := base
			if requestID, ok := xcontext.GetRequestID(r.Context()); ok {
				logger = logger.With(xslog.RequestID(requestID))
			}
			if sessionID, ok := xcontext.GetSessionID(r.Context()); ok {
				logger = logger.With(xslog.SessionID(sessionID))
			}
			ctx := xslog.WithLogger(r.Context(), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
