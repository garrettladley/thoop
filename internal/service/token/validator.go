package token

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/apperr"
	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/oauth2"
)

const defaultTokenCacheTTL = 5 * time.Minute

type Validator struct {
	cache        storage.TokenCache
	whoopLimiter storage.WhoopRateLimiter
	ttl          time.Duration
}

var _ Service = (*Validator)(nil)

func NewValidator(cache storage.TokenCache, whoopLimiter storage.WhoopRateLimiter) *Validator {
	return &Validator{
		cache:        cache,
		whoopLimiter: whoopLimiter,
		ttl:          defaultTokenCacheTTL,
	}
}

func (v *Validator) ValidateAndGetUserID(ctx context.Context, authHeader string) (int64, error) {
	logger := xslog.FromContext(ctx)

	token, err := extractBearerToken(authHeader)
	if err != nil {
		return 0, err
	}

	tokenHash := hashSecret(token)

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

type staticTokenSource struct {
	token string
}

var _ oauth2.TokenSource = (*staticTokenSource)(nil)

func (s *staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.token}, nil
}

func (v *Validator) validateWithWhoopAPI(ctx context.Context, token string, tokenHash string) (int64, error) {
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

		return 0, apperr.TooManyRequests("rate_limited", "rate limit exceeded", retryAfter, string(reason))
	}

	tokenSource := &staticTokenSource{token: token}
	client := whoop.New(tokenSource, whoop.WithTimeout(10*time.Second))

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

func hashSecret(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
