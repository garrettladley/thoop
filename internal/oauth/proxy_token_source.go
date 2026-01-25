package oauth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	go_json "github.com/goccy/go-json"

	"github.com/garrettladley/thoop/internal/keyring"
	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	"golang.org/x/oauth2"
)

var _ TokenSource = (*ProxyTokenSource)(nil)

// ProxyTokenSource implements oauth2.TokenSource and TokenChecker,
// refreshing tokens via the server's /auth/refresh endpoint.
// Secrets are stored in the OS keyring, metadata in SQLite.
type ProxyTokenSource struct {
	serverURL string
	querier   sqlitec.Querier
	keyring   keyring.Store
	client    *http.Client
	mu        sync.Mutex
	token     *oauth2.Token
}

func NewProxyTokenSource(serverURL string, querier sqlitec.Querier, kr keyring.Store) *ProxyTokenSource {
	return &ProxyTokenSource{
		serverURL: serverURL,
		querier:   querier,
		keyring:   kr,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *ProxyTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != nil && s.token.Valid() {
		return s.token, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := s.loadCredentials(ctx)
	if err != nil {
		return nil, err
	}

	if token.Valid() {
		s.token = token
		return token, nil
	}

	if token.RefreshToken == "" {
		return nil, ErrTokenExpired
	}

	// refresh via proxy
	newToken, err := s.refreshViaProxy(ctx, token.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if err := s.saveCredentials(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	s.token = newToken
	return newToken, nil
}

func (s *ProxyTokenSource) loadCredentials(ctx context.Context) (*oauth2.Token, error) {
	metadata, err := s.querier.GetTokenMetadata(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoToken
		}
		return nil, fmt.Errorf("failed to load token metadata: %w", err)
	}

	accessToken, err := s.keyring.Get(keyring.KeyAccessToken)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrNoToken
		}
		return nil, fmt.Errorf("failed to load access token from keyring: %w", err)
	}

	refreshToken, _ := s.keyring.Get(keyring.KeyRefreshToken) // optional

	return &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    metadata.TokenType,
		Expiry:       metadata.Expiry,
	}, nil
}

func (s *ProxyTokenSource) saveCredentials(ctx context.Context, token *oauth2.Token) error {
	if err := s.keyring.Set(keyring.KeyAccessToken, token.AccessToken); err != nil {
		return fmt.Errorf("failed to save access token to keyring: %w", err)
	}

	if token.RefreshToken != "" {
		if err := s.keyring.Set(keyring.KeyRefreshToken, token.RefreshToken); err != nil {
			return fmt.Errorf("failed to save refresh token to keyring: %w", err)
		}
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	params := sqlitec.UpsertTokenMetadataParams{
		TokenType: tokenType,
		Expiry:    token.Expiry,
	}
	if err := s.querier.UpsertTokenMetadata(ctx, params); err != nil {
		return fmt.Errorf("failed to save token metadata: %w", err)
	}

	return nil
}

func (s *ProxyTokenSource) refreshViaProxy(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	reqBody := struct {
		RefreshToken string `json:"refresh_token"`
	}{
		RefreshToken: refreshToken,
	}

	body, err := go_json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL+"/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var respBody struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := go_json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  respBody.AccessToken,
		TokenType:    respBody.TokenType,
		RefreshToken: respBody.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(respBody.ExpiresIn) * time.Second),
	}

	return token, nil
}

func (s *ProxyTokenSource) HasToken(ctx context.Context) (bool, error) {
	_, err := s.loadCredentials(ctx)
	if err != nil {
		if errors.Is(err, ErrNoToken) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *ProxyTokenSource) ExpiresWithin(ctx context.Context, d time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != nil {
		return time.Until(s.token.Expiry) <= d, nil
	}

	metadata, err := s.querier.GetTokenMetadata(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNoToken
		}
		return false, fmt.Errorf("failed to load token metadata: %w", err)
	}

	return time.Until(metadata.Expiry) <= d, nil
}

func (s *ProxyTokenSource) RefreshIfNeeded(ctx context.Context, threshold time.Duration) (*oauth2.Token, error) {
	expiresWithin, err := s.ExpiresWithin(ctx, threshold)
	if err != nil {
		return nil, err
	}
	if !expiresWithin {
		return nil, nil
	}

	return s.Token()
}

func (s *ProxyTokenSource) GetAPIKey() (string, error) {
	return s.keyring.Get(keyring.KeyAPIKey)
}

func (s *ProxyTokenSource) SaveAPIKey(apiKey string) error {
	return s.keyring.Set(keyring.KeyAPIKey, apiKey)
}
