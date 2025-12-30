package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/garrettladley/thoop/internal/service/user"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xerrors"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

// APIKeyAuth validates API keys and sets the verified user ID in context.
// Must be called before BearerAuth to enable the binding check.
func APIKeyAuth(userService user.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := xslog.FromContext(r.Context())

			apiKey := xhttp.GetRequestHeaderAPIKey(r)
			if apiKey == "" {
				logger.WarnContext(r.Context(), "missing API key header",
					xslog.RequestPath(r))
				xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("missing API key")))
				return
			}

			validatedUser, err := userService.ValidateAPIKey(r.Context(), apiKey)
			if err != nil {
				logger.WarnContext(r.Context(), "API key validation failed",
					xslog.RequestPath(r),
					xslog.ErrorGroup(err))

				switch {
				case errors.Is(err, user.ErrAPIKeyNotFound):
					xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("invalid API key")))
				case errors.Is(err, user.ErrAPIKeyRevoked):
					xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("API key has been revoked")))
				case errors.Is(err, user.ErrUserBanned):
					xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("account banned")))
				default:
					xerrors.WriteError(w, xerrors.Internal(xerrors.WithMessage("API key validation failed"), xerrors.WithCause(err)))
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
