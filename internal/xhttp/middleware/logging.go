package middleware

import (
	"net/http"
	"time"

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

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		xslog.FromContext(r.Context()).InfoContext(
			r.Context(),
			"http request",
			xslog.RequestGroup(r),
			xslog.ResponseGroup(wrapped.status, time.Since(start)),
		)
	})
}
