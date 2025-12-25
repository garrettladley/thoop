package oauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/garrettladley/thoop/internal/sqlc"
	"golang.org/x/oauth2"
)

type TokenChecker interface {
	HasToken(ctx context.Context) (bool, error)
}

var (
	_ TokenChecker       = (*DBTokenSource)(nil)
	_ oauth2.TokenSource = (*DBTokenSource)(nil)
)

type DBTokenSource struct {
	config  *oauth2.Config
	querier sqlc.Querier
	mu      sync.Mutex
	token   *oauth2.Token
}

func NewDBTokenSource(config *oauth2.Config, querier sqlc.Querier) *DBTokenSource {
	return &DBTokenSource{
		config:  config,
		querier: querier,
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

	src := s.config.TokenSource(ctx, token)

	newToken, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if err := s.saveToken(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	s.token = newToken

	return newToken, nil
}

func (s *DBTokenSource) HasToken(ctx context.Context) (bool, error) {
	_, err := s.querier.GetToken(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *DBTokenSource) saveToken(ctx context.Context, token *oauth2.Token) error {
	params := sqlc.UpsertTokenParams{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Expiry:      token.Expiry,
	}

	if token.RefreshToken != "" {
		params.RefreshToken = &token.RefreshToken
	}

	return s.querier.UpsertToken(ctx, params)
}

func dbTokenToOAuth2(t sqlc.Token) *oauth2.Token {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
		TokenType:   t.TokenType,
		Expiry:      t.Expiry,
	}

	if t.RefreshToken != nil {
		token.RefreshToken = *t.RefreshToken
	}

	return token
}

var (
	ErrNoToken      = errors.New("no token found - please authenticate first")
	ErrTokenExpired = errors.New("token expired and no refresh token available")
)
