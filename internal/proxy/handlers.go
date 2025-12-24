package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/garrettladley/thoop/internal/oauth"
	"golang.org/x/oauth2"
)

type stateEntry struct {
	localPort string
	createdAt time.Time
}

type Handler struct {
	config      *oauth2.Config
	states      map[string]stateEntry
	statesMu    sync.RWMutex
	stateTTL    time.Duration
	cleanupDone chan struct{}
}

func NewHandler(cfg Config) *Handler {
	h := &Handler{
		config:      oauth.NewConfig(cfg),
		states:      make(map[string]stateEntry),
		stateTTL:    10 * time.Minute,
		cleanupDone: make(chan struct{}),
	}

	go h.cleanupExpiredStates()

	return h
}

func (h *Handler) Close() {
	close(h.cleanupDone)
}

func (h *Handler) cleanupExpiredStates() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.statesMu.Lock()
			now := time.Now()
			for state, entry := range h.states {
				if now.Sub(entry.createdAt) > h.stateTTL {
					delete(h.states, state)
				}
			}
			h.statesMu.Unlock()
		case <-h.cleanupDone:
			return
		}
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

	h.statesMu.Lock()
	h.states[state] = stateEntry{
		localPort: localPort,
		createdAt: time.Now(),
	}
	h.statesMu.Unlock()

	authURL := h.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *Handler) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	h.statesMu.Lock()
	entry, ok := h.states[state]
	if ok {
		delete(h.states, state)
	}
	h.statesMu.Unlock()

	if !ok {
		http.Error(w, "invalid or expired state parameter", http.StatusBadRequest)
		return
	}

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		redirectWithError(w, r, entry.localPort, errParam, errDesc)
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

	redirectWithToken(w, r, entry.localPort, token)
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
