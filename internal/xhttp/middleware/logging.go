package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xslog"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			requestID, _ := xcontext.GetRequestID(r.Context())
			logger.InfoContext(r.Context(), "request",
				xslog.RequestID(requestID),
				xslog.RequestMethod(r),
				xslog.RequestPath(r),
				xslog.HTTPStatus(wrapped.status),
				xslog.Duration(time.Since(start)),
				xslog.RequestIP(r),
			)
		})
	}
}
