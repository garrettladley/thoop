package proxy

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

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
				xhttp.Error(w, http.StatusServiceUnavailable)
				return
			}

			if !result.Allowed {
				xhttp.WriteRateLimitError(w, &xhttp.RateLimitError{
					RetryAfter: result.RetryAfter,
					Reason:     "ip_rate_limit",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
