package token

import (
	"context"
	"errors"
)

var (
	ErrMissingToken = errors.New("missing bearer token")
	ErrInvalidToken = errors.New("invalid bearer token")
)

type Service interface {
	// ValidateAndGetUserID validates a bearer token from the Authorization header
	// and returns the associated WHOOP user ID.
	// Returns ErrMissingToken if the header is empty or malformed.
	// Returns ErrInvalidToken if the token is invalid or expired.
	ValidateAndGetUserID(ctx context.Context, authHeader string) (int64, error)
}
