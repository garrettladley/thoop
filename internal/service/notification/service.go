package notification

import (
	"context"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
)

type PollResult struct {
	Notifications []storage.Notification `json:"notifications"`
	ServerTime    time.Time              `json:"server_time"`
}

type Service interface {
	// GetNotifications retrieves notifications since a given timestamp.
	GetNotifications(ctx context.Context, userID int64, since time.Time) (*PollResult, error)

	// Subscribe creates a subscription for live notifications.
	// Returns a channel that receives notifications and an unsubscribe function.
	Subscribe(ctx context.Context, userID int64) (<-chan storage.Notification, func(), error)
}
