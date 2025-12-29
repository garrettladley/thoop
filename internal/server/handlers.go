package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/oauth"
	pgc "github.com/garrettladley/thoop/internal/sqlc/postgres"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/version"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

const stateTTL = 5 * time.Minute

type Handler struct {
	config  *oauth2.Config
	storage storage.StateStore
	db      pgc.Querier
}

func NewHandler(cfg Config, store storage.StateStore, db pgc.Querier) *Handler {
	return &Handler{
		config:  oauth.NewConfig(cfg),
		storage: store,
		db:      db,
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	token, err := h.config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Get user profile from WHOOP API
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token.AccessToken})
	whoopClient := whoop.New(tokenSource)

	profile, err := whoopClient.User.GetProfile(ctx)
	if err != nil {
		http.Error(w, "failed to get user profile", http.StatusInternalServerError)
		return
	}

	var apiKey string
	user, err := h.db.GetUser(ctx, profile.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		user, err = h.db.CreateUser(ctx, profile.UserID)
		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		apiKey, err = generateAPIKey()
		if err != nil {
			http.Error(w, "failed to generate API key", http.StatusInternalServerError)
			return
		}

		keyHash := hashToken(apiKey)
		keyName := "default"
		_, err = h.db.CreateAPIKey(ctx, pgc.CreateAPIKeyParams{
			WhoopUserID: profile.UserID,
			KeyHash:     keyHash,
			Name:        &keyName,
		})
		if err != nil {
			http.Error(w, "failed to create API key", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	if user.Banned {
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

const (
	apiKeyPrefix = "thp_"
	apiKeyLength = 32
)

func generateAPIKey() (string, error) {
	b := make([]byte, apiKeyLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return apiKeyPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}
