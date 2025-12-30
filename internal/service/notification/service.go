package notification

import (
	"context"

	"github.com/garrettladley/thoop/internal/storage"
)

type PollResult struct {
	Notifications []storage.Notification `json:"notifications"`
}

type Service interface {
	// GetUnacked retrieves unacknowledged notifications using cursor-based pagination.
	// Pass 0 for the first page. cursor is the monotonic id.
	GetUnacked(ctx context.Context, userID int64, cursor int64, limit int32) (*PollResult, error)

	// Acknowledge marks the specified notifications as acknowledged by trace_id.
	Acknowledge(ctx context.Context, userID int64, traceIDs []string) error

	// Subscribe creates a subscription for live notifications.
	// Returns a channel that receives notifications and an unsubscribe function.
	Subscribe(ctx context.Context, userID int64) (<-chan storage.Notification, func(), error)
}
