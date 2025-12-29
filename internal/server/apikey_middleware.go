package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/garrettladley/thoop/internal/service/user"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

// APIKeyAuth validates API keys and sets the verified user ID in context.
// Must be called before WhoopAuth to enable the binding check.
func APIKeyAuth(userService user.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := xslog.FromContext(r.Context())

			apiKey := xhttp.GetRequestHeaderAPIKey(r)
			if apiKey == "" {
				logger.WarnContext(r.Context(), "missing API key header",
					xslog.RequestPath(r))
				xhttp.SetHeaderContentTypeApplicationJSON(w)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"missing API key"}`))
				return
			}

			validatedUser, err := userService.ValidateAPIKey(r.Context(), apiKey)
			if err != nil {
				logger.WarnContext(r.Context(), "API key validation failed",
					xslog.RequestPath(r),
					xslog.ErrorGroup(err))

				xhttp.SetHeaderContentTypeApplicationJSON(w)

				switch {
				case errors.Is(err, user.ErrAPIKeyNotFound):
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized","message":"invalid API key"}`))
				case errors.Is(err, user.ErrAPIKeyRevoked):
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized","message":"API key has been revoked"}`))
				case errors.Is(err, user.ErrUserBanned):
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized","message":"account banned"}`))
				default:
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"internal_error","message":"API key validation failed"}`))
				}
				return
			}

			go func() {
				ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
				defer cancel()

				if err := userService.UpdateAPIKeyLastUsed(ctx, validatedUser.APIKeyID); err != nil {
					logger.WarnContext(ctx, "failed to update API key last_used_at",
						xslog.ErrorGroup(err))
				}
			}()

			ctx := xcontext.SetWhoopUserID(r.Context(), validatedUser.WhoopUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
