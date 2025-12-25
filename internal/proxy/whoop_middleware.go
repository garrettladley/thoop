package proxy

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/garrettladley/thoop/internal/xslog"
)

// WhoopAuth validates bearer tokens and sets the verified user ID in context.
// Validates tokens with WHOOP API (with caching) instead of trusting client headers.
func WhoopAuth(validator *TokenValidator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			userID, err := validator.ValidateAndGetUserID(r.Context(), authHeader)
			if err != nil {
				logger.WarnContext(r.Context(), "token validation failed",
					slog.String(keyPath, r.URL.Path),
					xslog.Error(err))

				w.Header().Set("Content-Type", "application/json")

				switch {
				case errors.Is(err, ErrMissingToken):
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Missing Authorization header"}`))
				case errors.Is(err, ErrInvalidToken):
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Invalid or expired token"}`))
				default:
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"internal_error","message":"Token validation failed"}`))
				}
				return
			}

			ctx := SetWhoopUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
