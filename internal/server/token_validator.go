package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/oauth2"
)

var (
	ErrMissingToken = errors.New("missing bearer token")
	ErrInvalidToken = errors.New("invalid bearer token")
)

const defaultTokenCacheTTL = 5 * time.Minute

type TokenValidator struct {
	cache        storage.TokenCache
	whoopLimiter storage.WhoopRateLimiter
	httpClient   *http.Client
	ttl          time.Duration
}

func NewTokenValidator(cache storage.TokenCache, whoopLimiter storage.WhoopRateLimiter) *TokenValidator {
	return &TokenValidator{
		cache:        cache,
		whoopLimiter: whoopLimiter,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		ttl:          defaultTokenCacheTTL,
	}
}

func (v *TokenValidator) ValidateAndGetUserID(ctx context.Context, authHeader string) (int64, error) {
	logger := xslog.FromContext(ctx)

	token, err := extractBearerToken(authHeader)
	if err != nil {
		return 0, err
	}

	tokenHash := hashToken(token)

	userID, err := v.cache.GetUserID(ctx, tokenHash)
	if err == nil {
		logger.DebugContext(ctx, "token cache hit")
		return userID, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		logger.WarnContext(ctx, "token cache error", xslog.ErrorGroup(err))
	}

	logger.DebugContext(ctx, "token cache miss, validating with WHOOP API")

	userID, err = v.validateWithWhoopAPI(ctx, token, tokenHash)
	if err != nil {
		return 0, err
	}

	if cacheErr := v.cache.SetUserID(ctx, tokenHash, userID, v.ttl); cacheErr != nil {
		logger.WarnContext(ctx, "failed to cache token", xslog.ErrorGroup(cacheErr))
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

func (v *TokenValidator) validateWithWhoopAPI(ctx context.Context, token string, tokenHash string) (int64, error) {
	logger := xslog.FromContext(ctx)

	state, err := v.whoopLimiter.CheckAndIncrement(ctx, tokenHash)
	if err != nil {
		return 0, fmt.Errorf("checking rate limit: %w", err)
	}
	if !state.Allowed {
		logger.WarnContext(ctx, "rate limit exceeded for token validation")

		var retryAfter time.Duration
		var reason storage.WhoopRateLimitReason
		if state.Reason != nil {
			reason = *state.Reason
			switch reason {
			case storage.WhoopRateLimitReasonPerUserMinute, storage.WhoopRateLimitReasonGlobalMinute:
				retryAfter = time.Until(state.MinuteReset)
			case storage.WhoopRateLimitReasonPerUserDay, storage.WhoopRateLimitReasonGlobalDay:
				retryAfter = time.Until(state.DayReset)
			default:
				retryAfter = time.Minute
			}
		} else {
			retryAfter = time.Minute
		}

		return 0, &xhttp.RateLimitError{RetryAfter: retryAfter, Reason: string(reason)}
	}

	tokenSource := &staticTokenSource{token: token}
	client := whoop.New(tokenSource, whoop.WithHTTPClient(v.httpClient))

	profile, err := client.User.GetProfile(ctx)
	if err != nil {
		var apiErr *whoop.APIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode == http.StatusUnauthorized {
				return 0, ErrInvalidToken
			}
		}
		return 0, fmt.Errorf("validating token: %w", err)
	}

	return profile.UserID, nil
}
