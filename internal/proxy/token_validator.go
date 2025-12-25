package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/oauth2"
)

var (
	ErrMissingToken = errors.New("missing bearer token")
	ErrInvalidToken = errors.New("invalid bearer token")
)

const defaultTokenCacheTTL = 5 * time.Minute

type TokenValidator struct {
	cache      storage.TokenCache
	httpClient *http.Client
	logger     *slog.Logger
	ttl        time.Duration
}

func NewTokenValidator(cache storage.TokenCache, logger *slog.Logger) *TokenValidator {
	return &TokenValidator{
		cache:      cache,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		ttl:        defaultTokenCacheTTL,
	}
}

func (v *TokenValidator) ValidateAndGetUserID(ctx context.Context, authHeader string) (string, error) {
	token, err := extractBearerToken(authHeader)
	if err != nil {
		return "", err
	}

	tokenHash := hashToken(token)

	userID, err := v.cache.GetUserID(ctx, tokenHash)
	if err == nil {
		v.logger.DebugContext(ctx, "token cache hit")
		return userID, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		v.logger.WarnContext(ctx, "token cache error", xslog.Error(err))
	}

	v.logger.DebugContext(ctx, "token cache miss, validating with WHOOP API")

	userID, err = v.validateWithWhoopAPI(ctx, token)
	if err != nil {
		return "", err
	}

	if cacheErr := v.cache.SetUserID(ctx, tokenHash, userID, v.ttl); cacheErr != nil {
		v.logger.WarnContext(ctx, "failed to cache token", xslog.Error(cacheErr))
	}

	return userID, nil
}

func extractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrMissingToken
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", ErrInvalidToken
	}

	token := strings.TrimPrefix(authHeader, prefix)
	if token == "" {
		return "", ErrInvalidToken
	}

	return token, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type staticTokenSource struct {
	token string
}

var _ oauth2.TokenSource = (*staticTokenSource)(nil)

func (s *staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.token}, nil
}

func (v *TokenValidator) validateWithWhoopAPI(ctx context.Context, token string) (string, error) {
	tokenSource := &staticTokenSource{token: token}
	client := whoop.New(tokenSource, whoop.WithHTTPClient(v.httpClient))

	profile, err := client.User.GetProfile(ctx)
	if err != nil {
		var apiErr *whoop.APIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode == http.StatusUnauthorized {
				return "", ErrInvalidToken
			}
		}
		return "", fmt.Errorf("validating token: %w", err)
	}

	return strconv.FormatInt(profile.UserID, 10), nil
}
