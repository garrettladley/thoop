package user

import (
	"context"
	"errors"
)

var (
	ErrAPIKeyNotFound = errors.New("API key not found")
	ErrAPIKeyRevoked  = errors.New("API key has been revoked")
	ErrUserBanned     = errors.New("user account is banned")
	ErrUserNotFound   = errors.New("user not found")
)

type ValidatedUser struct {
	WhoopUserID int64
	APIKeyID    int64
}

type Service interface {
	// ValidateAPIKey validates an API key and returns the associated user info.
	// Returns ErrAPIKeyNotFound if the key doesn't exist,
	// ErrAPIKeyRevoked if the key has been revoked,
	// or ErrUserBanned if the user account is banned.
	ValidateAPIKey(ctx context.Context, apiKey string) (*ValidatedUser, error)

	// GetOrCreateUser retrieves an existing user or creates a new one with an API key.
	// For new users, returns the plaintext API key (only time it's available).
	// For existing users, apiKey will be empty.
	GetOrCreateUser(ctx context.Context, whoopUserID int64) (apiKey string, banned bool, err error)

	// UpdateAPIKeyLastUsed updates the last_used_at timestamp for an API key.
	// This is typically called asynchronously after successful validation.
	UpdateAPIKeyLastUsed(ctx context.Context, apiKeyID int64) error

	// IsBanned checks if a user is banned.
	IsBanned(ctx context.Context, whoopUserID int64) (bool, error)
}
