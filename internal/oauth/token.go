package oauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/garrettladley/thoop/internal/keyring"
	litesqlc "github.com/garrettladley/thoop/internal/sqlite/sqlc"
	"golang.org/x/oauth2"
)

type TokenChecker interface {
	HasToken(ctx context.Context) (bool, error)
}

// TokenSource combines oauth2.TokenSource with token management capabilities.
type TokenSource interface {
	oauth2.TokenSource
	TokenChecker
	RefreshIfNeeded(ctx context.Context, threshold time.Duration) (*oauth2.Token, error)
}

var _ TokenSource = (*DBTokenSource)(nil)

// DBTokenSource implements TokenSource using SQLite for metadata and OS keyring for secrets.
type DBTokenSource struct {
	config  *oauth2.Config
	querier litesqlc.Querier
	keyring keyring.Store
	mu      sync.Mutex
	token   *oauth2.Token
}

func NewDBTokenSource(config *oauth2.Config, querier litesqlc.Querier, kr keyring.Store) *DBTokenSource {
	return &DBTokenSource{
		config:  config,
		querier: querier,
		keyring: kr,
	}
}

func (s *DBTokenSource) Token() (*oauth2.Token, error) {
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

	src := s.config.TokenSource(ctx, token)

	newToken, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if err := s.saveCredentials(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	s.token = newToken

	return newToken, nil
}

func (s *DBTokenSource) loadCredentials(ctx context.Context) (*oauth2.Token, error) {
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

	refreshToken, _ := s.keyring.Get(keyring.KeyRefreshToken)

	return &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    metadata.TokenType,
		Expiry:       metadata.Expiry,
	}, nil
}

func (s *DBTokenSource) saveCredentials(ctx context.Context, token *oauth2.Token) error {
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
		tokenType = defaultTokenType
	}

	params := litesqlc.UpsertTokenMetadataParams{
		TokenType: tokenType,
		Expiry:    token.Expiry,
	}
	if err := s.querier.UpsertTokenMetadata(ctx, params); err != nil {
		return fmt.Errorf("failed to save token metadata: %w", err)
	}

	return nil
}

func (s *DBTokenSource) HasToken(ctx context.Context) (bool, error) {
	_, err := s.loadCredentials(ctx)
	if err != nil {
		if errors.Is(err, ErrNoToken) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check token: %w", err)
	}
	return true, nil
}

func (s *DBTokenSource) ExpiresWithin(ctx context.Context, d time.Duration) (bool, error) {
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

func (s *DBTokenSource) RefreshIfNeeded(ctx context.Context, threshold time.Duration) (*oauth2.Token, error) {
	expiresWithin, err := s.ExpiresWithin(ctx, threshold)
	if err != nil {
		return nil, err
	}
	if !expiresWithin {
		return nil, nil
	}

	return s.Token()
}

func (s *DBTokenSource) GetAPIKey() (string, error) {
	apiKey, err := s.keyring.Get(keyring.KeyAPIKey)
	if err != nil {
		return "", fmt.Errorf("failed to get API key from keyring: %w", err)
	}
	return apiKey, nil
}

func (s *DBTokenSource) SaveAPIKey(apiKey string) error {
	if err := s.keyring.Set(keyring.KeyAPIKey, apiKey); err != nil {
		return fmt.Errorf("failed to save API key to keyring: %w", err)
	}
	return nil
}

var (
	ErrNoToken      = errors.New("no token found - please authenticate first")
	ErrTokenExpired = errors.New("token expired and no refresh token available")
)
