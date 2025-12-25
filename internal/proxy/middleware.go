package proxy

import (
	"log/slog"
	"net/http"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

const (
	keyRequestID     = "request_id"
	keyMethod        = "method"
	keyPath          = "path"
	keyStatus        = "status"
	keyDuration      = "duration"
	keyIP            = "ip"
	keyError         = "error"
	keyStack         = "stack"
	keyWhoopUserID   = "whoop_user_id"
	keyURL           = "url"
	keyClientVersion = "client_version"
	keyProxyVersion  = "proxy_version"
	keyMinVersion    = "min_version"
)

func RateLimitWithBackend(backend storage.RateLimiter, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := xhttp.GetRequestIP(r)

			allowed, err := backend.Allow(r.Context(), ip)
			if err != nil {
				logger.ErrorContext(r.Context(), "rate limit check failed",
					xslog.Error(err),
					xslog.IP(ip),
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
