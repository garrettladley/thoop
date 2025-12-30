package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/service/auth"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/oauth2"
)

type Auth struct {
	service auth.Service
}

func NewAuth(service auth.Service) *Auth {
	return &Auth{service: service}
}

// HandleAuthStart handles GET /auth/start requests.
func (h *Auth) HandleAuthStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	localPort := r.URL.Query().Get(oauth.ParamLocalPort)

	req := auth.StartAuthRequest{
		LocalPort:     localPort,
		ClientVersion: r.URL.Query().Get(oauth.ParamClientVersion),
	}

	result, err := h.service.StartAuth(ctx, req)
	if err != nil {
		// handle version error - redirect with version info
		var verr *auth.VersionError
		if errors.As(err, &verr) {
			redirectWithError(w, r, localPort,
				oauth.ErrorCodeIncompatibleVersion,
				verr.Error(),
				map[string]string{oauth.ParamMinVersion: verr.MinVersion},
			)
			return
		}

		// handle invalid port
		if errors.Is(err, auth.ErrInvalidPort) {
			http.Error(w, "invalid local_port parameter", http.StatusBadRequest)
			return
		}

		// generic error
		http.Error(w, "failed to start auth", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, result.AuthURL, http.StatusTemporaryRedirect)
}

// HandleAuthCallback handles GET /auth/callback requests.
func (h *Auth) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	req := auth.CallbackRequest{
		State:     r.URL.Query().Get("state"),
		Code:      r.URL.Query().Get("code"),
		ErrorCode: r.URL.Query().Get(oauth.ParamError),
		ErrorDesc: r.URL.Query().Get(oauth.ParamErrorDescription),
	}

	result, err := h.service.HandleCallback(ctx, req)
	if err != nil {
		// handle AuthError - redirect with error info
		var authErr *auth.AuthError
		if errors.As(err, &authErr) {
			redirectWithError(w, r, authErr.LocalPort,
				oauth.ErrorCode(authErr.ErrorCode),
				authErr.ErrorDesc,
				authErr.Extra,
			)
			return
		}

		// handle state errors without local port
		if errors.Is(err, auth.ErrInvalidState) {
			http.Error(w, "invalid or expired state parameter", http.StatusBadRequest)
			return
		}

		logger.ErrorContext(ctx, "auth callback error", xslog.Error(err))
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	redirectWithToken(w, r, result.LocalPort, result.Token, result.APIKey)
}

func redirectWithToken(w http.ResponseWriter, r *http.Request, localPort string, token *oauth2.Token, apiKey string) {
	callbackURL := fmt.Sprintf("http://localhost:%s/callback", localPort)

	u, _ := url.Parse(callbackURL)
	q := u.Query()

	q.Set("access_token", token.AccessToken)
	q.Set("token_type", token.TokenType)
	q.Set("expires_in", fmt.Sprintf("%d", int(time.Until(token.Expiry).Seconds())))
	if token.RefreshToken != "" {
		q.Set("refresh_token", token.RefreshToken)
	}

	if apiKey != "" {
		q.Set("api_key", apiKey)
	}
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

func redirectWithError(w http.ResponseWriter, r *http.Request, localPort string, errCode oauth.ErrorCode, errDesc string, extra map[string]string) {
	callbackURL := fmt.Sprintf("http://localhost:%s/callback", localPort)

	u, _ := url.Parse(callbackURL)
	q := u.Query()
	q.Set(oauth.ParamError, string(errCode))
	q.Set(oauth.ParamErrorDescription, errDesc)

	for k, v := range extra {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}
