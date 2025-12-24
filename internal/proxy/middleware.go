package proxy

import (
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"slices"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/google/uuid"
)

const (
	headerXForwardedFor    = "X-Forwarded-For"
	headerXRequestID       = "X-Request-ID"
	headerXContentTypeOpts = "X-Content-Type-Options"
	headerXFrameOpts       = "X-Frame-Options"
	headerXXSSProtection   = "X-XSS-Protection"
	headerReferrerPolicy   = "Referrer-Policy"
)

const (
	keyRequestID = "request_id"
	keyMethod    = "method"
	keyPath      = "path"
	keyStatus    = "status"
	keyDuration  = "duration"
	keyIP        = "ip"
	keyError     = "error"
	keyStack     = "stack"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		ctx := SetRequestID(r.Context(), id)
		w.Header().Set(headerXRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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

			requestID, _ := GetRequestID(r.Context())
			logger.InfoContext(r.Context(), "request",
				slog.String(keyRequestID, requestID),
				slog.String(keyMethod, r.Method),
				slog.String(keyPath, r.URL.Path),
				slog.Int(keyStatus, wrapped.status),
				slog.Duration(keyDuration, time.Since(start)),
				slog.String(keyIP, getIP(r)),
			)
		})
	}
}

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID, _ := GetRequestID(r.Context())
					logger.ErrorContext(r.Context(), "panic recovered",
						slog.String(keyRequestID, requestID),
						slog.Any(keyError, err),
						slog.String(keyStack, string(debug.Stack())),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerXContentTypeOpts, "nosniff")
		w.Header().Set(headerXFrameOpts, "DENY")
		w.Header().Set(headerXXSSProtection, "1; mode=block")
		w.Header().Set(headerReferrerPolicy, "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func RateLimitWithBackend(backend storage.RateLimiter, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			allowed, err := backend.Allow(r.Context(), ip)
			if err != nil {
				logger.ErrorContext(r.Context(), "rate limit check failed",
					slog.Any(keyError, err),
					slog.String(keyIP, ip),
				)
				http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
				return
			}

			if !allowed {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) string {
	if xff := r.Header.Get(headerXForwardedFor); xff != "" {
		if ip, _, err := net.SplitHostPort(xff); err == nil {
			return ip
		}
		return xff
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

func Chain(h http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	slices.Reverse(middleware)
	for _, m := range middleware {
		h = m(h)
	}
	return h
}
