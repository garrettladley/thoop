package proxy

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

			userID, err := validator.ValidateAndGetUserID(r.Context(), authHeader)
			if err != nil {
				logger.WarnContext(r.Context(), "token validation failed",
					xslog.RequestPath(r),
					xslog.ErrorGroup(err))

				xhttp.SetHeaderContentTypeApplicationJSON(w)

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

			ctx := xcontext.SetWhoopUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
