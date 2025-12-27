package middleware

import (
	"log/slog"
	"net/http"

	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xslog"
)

// Logger injects an enriched logger into request context.
// Must run AFTER RequestID middleware.
func Logger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := base
			if id, ok := xcontext.GetRequestID(r.Context()); ok {
				logger = logger.With(xslog.RequestID(id))
			}
			ctx := xslog.WithLogger(r.Context(), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
