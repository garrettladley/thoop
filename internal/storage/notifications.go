package storage

import (
	"context"
	"time"
)

type EntityType string

const (
	EntityTypeRecovery EntityType = "recovery"
	EntityTypeWorkout  EntityType = "workout"
	EntityTypeSleep    EntityType = "sleep"
)

type Action string

const (
	ActionUpdated Action = "updated"
	ActionDeleted Action = "deleted"
)

type Notification struct {
	ID         int64      `json:"id"`       // monotonic, for cursor
	TraceID    string     `json:"trace_id"` // WHOOP's trace_id, for acking
	EntityType EntityType `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	Action     Action     `json:"action"`
	Timestamp  time.Time  `json:"timestamp"`
}

type NotificationStore interface {
	// Add persists a notification and publishes to real-time subscribers.
	// Idempotent: duplicate trace_ids are ignored.
	Add(ctx context.Context, userID int64, n Notification) error

	// GetUnacked returns unacknowledged notifications using cursor-based pagination.
	// Pass 0 for the first page. Returns notifications with id > cursor.
	GetUnacked(ctx context.Context, userID int64, cursor int64, limit int32) ([]Notification, error)

	// Acknowledge marks the specified notifications as acknowledged by trace_id.
	Acknowledge(ctx context.Context, userID int64, traceIDs []string) error

	// Subscribe returns a channel that receives notifications for a user.
	// The returned function should be called to unsubscribe.
	Subscribe(ctx context.Context, userID int64) (<-chan Notification, func(), error)
}
