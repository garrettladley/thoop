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

var _ http.Flusher = (*responseWriter)(nil)

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
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
