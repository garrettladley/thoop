package auth

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
)

var (
	ErrInvalidPort         = errors.New("invalid local port")
	ErrInvalidState        = errors.New("invalid or expired state")
	ErrIncompatibleVersion = errors.New("incompatible client version")
	ErrRateLimited         = errors.New("rate limit exceeded")
	ErrAccountBanned       = errors.New("account banned")
	ErrAuthDenied          = errors.New("authorization denied")
	ErrInvalidRefreshToken = errors.New("invalid or missing refresh token")
	ErrRefreshFailed       = errors.New("token refresh failed")
)

type StartAuthRequest struct {
	LocalPort     string
	ClientVersion string
}

type StartAuthResult struct {
	AuthURL string
}

type CallbackRequest struct {
	State     string
	Code      string
	ErrorCode string
	ErrorDesc string
}

type CallbackResult struct {
	Token     *oauth2.Token
	APIKey    string
	LocalPort string
}

type RefreshRequest struct {
	RefreshToken string
}

type RefreshResult struct {
	Token *oauth2.Token
}

type VersionError struct {
	MinVersion string
}

func (e *VersionError) Error() string {
	return "incompatible client version"
}

func (e *VersionError) Unwrap() error {
	return ErrIncompatibleVersion
}

type AuthError struct {
	Err       error
	LocalPort string
	ErrorCode string
	ErrorDesc string
	Extra     map[string]string
}

func (e *AuthError) Error() string {
	return e.Err.Error()
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

type Service interface {
	// StartAuth validates the request, stores state, and returns the auth URL.
	// Returns ErrInvalidPort if the local port is invalid.
	// Returns *VersionError (wrapping ErrIncompatibleVersion) if version check fails.
	StartAuth(ctx context.Context, req StartAuthRequest) (*StartAuthResult, error)

	// HandleCallback processes the OAuth callback.
	// Returns ErrInvalidState if state is missing or invalid.
	// Returns ErrAuthDenied if WHOOP denied authorization.
	// Returns ErrRateLimited if rate limited.
	// Returns ErrAccountBanned if the user is banned.
	// The returned *AuthError contains LocalPort for redirect construction.
	HandleCallback(ctx context.Context, req CallbackRequest) (*CallbackResult, error)

	// RefreshToken exchanges a refresh token for new access and refresh tokens.
	// Returns ErrInvalidRefreshToken if refresh token is missing.
	// Returns ErrRefreshFailed if the token exchange fails.
	RefreshToken(ctx context.Context, req RefreshRequest) (*RefreshResult, error)
}
