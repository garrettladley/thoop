package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/storage"
	"golang.org/x/oauth2"
)

const stateTTL = 5 * time.Minute

type Handler struct {
	config  *oauth2.Config
	storage storage.StateStore
}

func NewHandler(cfg Config, store storage.StateStore) *Handler {
	return &Handler{
		config:  oauth.NewConfig(cfg),
		storage: store,
	}
}

func (h *Handler) HandleAuthStart(w http.ResponseWriter, r *http.Request) {
	localPort := r.URL.Query().Get("local_port")
	if !isValidPort(localPort) {
		http.Error(w, "invalid local_port parameter", http.StatusBadRequest)
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	entry := storage.StateEntry{
		LocalPort: localPort,
		CreatedAt: time.Now(),
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

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		redirectWithError(w, r, entry.LocalPort, errParam, errDesc)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := h.config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	redirectWithToken(w, r, entry.LocalPort, token)
}

func redirectWithToken(w http.ResponseWriter, r *http.Request, localPort string, token *oauth2.Token) {
	callbackURL := fmt.Sprintf("http://localhost:%s/callback", localPort)

	u, _ := url.Parse(callbackURL)
	q := u.Query()
	q.Set("access_token", token.AccessToken)
	q.Set("token_type", token.TokenType)
	q.Set("expires_in", fmt.Sprintf("%d", int(time.Until(token.Expiry).Seconds())))
	if token.RefreshToken != "" {
		q.Set("refresh_token", token.RefreshToken)
	}
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

func redirectWithError(w http.ResponseWriter, r *http.Request, localPort string, errCode string, errDesc string) {
	callbackURL := fmt.Sprintf("http://localhost:%s/callback", localPort)

	u, _ := url.Parse(callbackURL)
	q := u.Query()
	q.Set("error", errCode)
	q.Set("error_description", errDesc)
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
