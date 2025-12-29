package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/service/user"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/version"
	"golang.org/x/oauth2"
)

const stateTTL = 5 * time.Minute

type Handler struct {
	config       *oauth2.Config
	storage      storage.StateStore
	userService  user.Service
	whoopLimiter storage.WhoopRateLimiter
}

func NewHandler(cfg Config, store storage.StateStore, userService user.Service, whoopLimiter storage.WhoopRateLimiter) *Handler {
	return &Handler{
		config:       oauth.NewConfig(cfg),
		storage:      store,
		userService:  userService,
		whoopLimiter: whoopLimiter,
	}
}

func (h *Handler) HandleAuthStart(w http.ResponseWriter, r *http.Request) {
	localPort := r.URL.Query().Get(oauth.ParamLocalPort)
	if !isValidPort(localPort) {
		http.Error(w, "invalid local_port parameter", http.StatusBadRequest)
		return
	}

	clientVersion := r.URL.Query().Get(oauth.ParamClientVersion)
	if clientVersion == "" {
		clientVersion = "unknown"
	}

	if verr := version.CheckCompatibility(clientVersion); verr != nil {
		redirectWithError(w, r, localPort, oauth.ErrorCodeIncompatibleVersion, verr.Error(), map[string]string{
			oauth.ParamMinVersion: verr.MinVersion,
		})
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	entry := storage.StateEntry{
		LocalPort:     localPort,
		ClientVersion: clientVersion,
		CreatedAt:     time.Now(),
	}

	if err := h.storage.Set(r.Context(), state, entry, stateTTL); err != nil {
		http.Error(w, "failed to store state", http.StatusInternalServerError)
		return
	}

	authURL := h.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *Handler) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	entry, err := h.storage.GetAndDelete(r.Context(), state)
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, "invalid or expired state parameter", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "failed to retrieve state", http.StatusInternalServerError)
		return
	}

	if errParam := r.URL.Query().Get(oauth.ParamError); errParam != "" {
		errDesc := r.URL.Query().Get(oauth.ParamErrorDescription)
		redirectWithError(w, r, entry.LocalPort, oauth.ErrorCode(errParam), errDesc, nil)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	token, err := h.config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	rlState, err := h.whoopLimiter.CheckAndIncrement(ctx, "oauth")
	if err != nil {
		http.Error(w, "failed to check rate limit", http.StatusInternalServerError)
		return
	}
	if !rlState.Allowed {
		redirectWithError(w, r, entry.LocalPort, oauth.ErrorCodeRateLimited, "rate limit exceeded, please try again later", nil)
		return
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token.AccessToken})
	whoopClient := whoop.New(tokenSource)

	profile, err := whoopClient.User.GetProfile(ctx)
	if err != nil {
		http.Error(w, "failed to get user profile", http.StatusInternalServerError)
		return
	}

	apiKey, banned, err := h.userService.GetOrCreateUser(ctx, profile.UserID)
	if err != nil {
		http.Error(w, "failed to get or create user", http.StatusInternalServerError)
		return
	}

	if banned {
		redirectWithError(w, r, entry.LocalPort, oauth.ErrorCodeAccountBanned, "your account has been banned", nil)
		return
	}

	redirectWithToken(w, r, entry.LocalPort, token, apiKey)
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

func isValidPort(s string) bool {
	if s == "" {
		return false
	}
	port, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return port >= 1 && port <= 65535
}
