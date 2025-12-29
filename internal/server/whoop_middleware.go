package server

import (
	"errors"
	"net/http"

	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

// WhoopAuth validates bearer tokens and sets the verified user ID in context.
// Validates tokens with WHOOP API (with caching) instead of trusting client headers.
func WhoopAuth(validator *TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := xslog.FromContext(r.Context())
			authHeader := r.Header.Get("Authorization")

			tokenUserID, err := validator.ValidateAndGetUserID(r.Context(), authHeader)
			if err != nil {
				logger.WarnContext(r.Context(), "token validation failed",
					xslog.RequestPath(r),
					xslog.ErrorGroup(err))

				xhttp.SetHeaderContentTypeApplicationJSON(w)

				var rateLimitErr *xhttp.RateLimitError
				switch {
				case errors.Is(err, ErrMissingToken):
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Missing Authorization header"}`))
				case errors.Is(err, ErrInvalidToken):
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Invalid or expired token"}`))
				case errors.As(err, &rateLimitErr):
					xhttp.WriteRateLimitError(w, rateLimitErr)
				default:
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"internal_error","message":"Token validation failed"}`))
				}
				return
			}

			// binding check: verify token's user matches API key's user (if APIKeyAuth ran first)
			if apiKeyUserID, ok := xcontext.GetWhoopUserID(r.Context()); ok {
				if tokenUserID != apiKeyUserID {
					logger.WarnContext(r.Context(), "token user does not match API key user",
						xslog.RequestPath(r),
						xslog.BindingMismatchGroup(tokenUserID, apiKeyUserID))
					xhttp.SetHeaderContentTypeApplicationJSON(w)
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"error":"forbidden","message":"token does not match API key"}`))
					return
				}
			}

			ctx := xcontext.SetWhoopUserID(r.Context(), tokenUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
