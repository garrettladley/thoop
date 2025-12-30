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

	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	"golang.org/x/oauth2"
)

var _ TokenSource = (*ProxyTokenSource)(nil)

// ProxyTokenSource implements oauth2.TokenSource and TokenChecker,
// refreshing tokens via the server's /auth/refresh endpoint.
type ProxyTokenSource struct {
	serverURL string
	querier   sqlitec.Querier
	client    *http.Client
	mu        sync.Mutex
	token     *oauth2.Token
}

func NewProxyTokenSource(serverURL string, querier sqlitec.Querier) *ProxyTokenSource {
	return &ProxyTokenSource{
		serverURL: serverURL,
		querier:   querier,
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

	dbToken, err := s.querier.GetToken(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoToken
		}
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	token := dbTokenToOAuth2(dbToken)

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

	if err := s.saveToken(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	s.token = newToken
	return newToken, nil
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
	_, err := s.querier.GetToken(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get token: %w", err)
	}
	return true, nil
}

func (s *ProxyTokenSource) ExpiresWithin(ctx context.Context, d time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != nil {
		return time.Until(s.token.Expiry) <= d, nil
	}

	dbToken, err := s.querier.GetToken(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNoToken
		}
		return false, fmt.Errorf("failed to load token: %w", err)
	}

	return time.Until(dbToken.Expiry) <= d, nil
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

func (s *ProxyTokenSource) saveToken(ctx context.Context, token *oauth2.Token) error {
	params := sqlitec.UpsertTokenParams{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Expiry:      token.Expiry,
	}

	if token.RefreshToken != "" {
		params.RefreshToken = &token.RefreshToken
	}

	err := s.querier.UpsertToken(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert token: %w", err)
	}
	return nil
}
