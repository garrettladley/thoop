package middleware

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/apperr"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

// RateLimitWithBackend applies IP-based rate limiting.
func RateLimitWithBackend(backend storage.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := xslog.FromContext(r.Context())
			ip := xhttp.GetRequestIP(r)

			result, err := backend.Allow(r.Context(), ip)
			if err != nil {
				logger.ErrorContext(r.Context(), "rate limit check failed",
					xslog.ErrorGroup(err),
					xslog.IP(ip),
				)
				apperr.WriteError(w, apperr.ServiceUnavailable("service_unavailable", "rate limit check failed"))
				return
			}

			if !result.Allowed {
				apperr.WriteError(w, apperr.TooManyRequests("rate_limited", "rate limit exceeded", result.RetryAfter, "ip_rate_limit"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
