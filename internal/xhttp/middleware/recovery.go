package middleware

import (
	"log/slog"
	"net/http"

	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xslog"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID, _ := xcontext.GetRequestID(r.Context())
					logger.ErrorContext(
						r.Context(),
						"panic recovered",
						xslog.RequestID(requestID),
						xslog.ErrorAny(err),
						xslog.Stack(),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
