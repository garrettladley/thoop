package middleware

import (
	"errors"
	"net/http"

	"github.com/garrettladley/thoop/internal/service/token"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xerrors"
	"github.com/garrettladley/thoop/internal/xslog"
)

// BearerAuth validates bearer tokens and sets the verified user ID in context.
// Validates tokens with WHOOP API instead of trusting client headers.
func BearerAuth(tokenService token.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := xslog.FromContext(r.Context())
			authHeader := r.Header.Get("Authorization")

			tokenUserID, err := tokenService.ValidateAndGetUserID(r.Context(), authHeader)
			if err != nil {
				logger.WarnContext(r.Context(), "token validation failed",
					xslog.RequestPath(r),
					xslog.ErrorGroup(err))

				switch {
				case errors.Is(err, token.ErrMissingToken):
					xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("missing Authorization header")))
				case errors.Is(err, token.ErrInvalidToken):
					xerrors.WriteError(w, xerrors.Unauthorized(xerrors.WithMessage("invalid or expired token")))
				default:
					if appErr := xerrors.As(err); appErr != nil && appErr.RateLimit != nil {
						xerrors.WriteError(w, appErr)
						return
					}
					xerrors.WriteError(w, xerrors.Internal(xerrors.WithMessage("token validation failed"), xerrors.WithCause(err)))
				}
				return
			}

			// binding check: verify token's user matches API key's user (if APIKeyAuth ran first)
			if apiKeyUserID, ok := xcontext.GetWhoopUserID(r.Context()); ok {
				if tokenUserID != apiKeyUserID {
					logger.WarnContext(r.Context(), "token user does not match API key user",
						xslog.RequestPath(r),
						xslog.BindingMismatchGroup(tokenUserID, apiKeyUserID))
					xerrors.WriteError(w, xerrors.Forbidden(xerrors.WithMessage("token does not match API key")))
					return
				}
			}

			ctx := xcontext.SetWhoopUserID(r.Context(), tokenUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
