package webhook

import (
	"context"
	"errors"
)

var (
	ErrMissingSignature = errors.New("missing signature headers")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrTimestampExpired = errors.New("timestamp too old")
	ErrUnknownEventType = errors.New("unknown event type")
)

type ProcessRequest struct {
	Body      []byte
	Signature string
	Timestamp string
}

type Service interface {
	// ProcessWebhook verifies the webhook signature, parses the event,
	// and stores/publishes the resulting notification.
	// Returns ErrMissingSignature if signature headers are empty.
	// Returns ErrInvalidSignature if the signature doesn't match.
	// Returns ErrTimestampExpired if the timestamp is too old.
	// Returns ErrUnknownEventType for unknown event types (caller may treat as success).
	ProcessWebhook(ctx context.Context, req ProcessRequest) error
}
