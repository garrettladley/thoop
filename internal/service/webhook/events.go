package webhook

import (
	"fmt"
	"strings"

	"github.com/garrettladley/thoop/internal/storage"
	go_json "github.com/goccy/go-json"
)

type Event interface {
	webhookEvent()
	GetUserID() int64
	GetAction() storage.Action
	GetEntityType() storage.EntityType
	GetEntityID() string
}

type eventBase struct {
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
}

func (e eventBase) GetUserID() int64 { return e.UserID }

type WorkoutEvent struct {
	eventBase
	WorkoutID string `json:"id"`
	action    storage.Action
}

func (e WorkoutEvent) webhookEvent()                     {}
func (e WorkoutEvent) GetAction() storage.Action         { return e.action }
func (e WorkoutEvent) GetEntityType() storage.EntityType { return storage.EntityTypeWorkout }
func (e WorkoutEvent) GetEntityID() string               { return e.WorkoutID }

type SleepEvent struct {
	eventBase
	SleepID string `json:"id"`
	action  storage.Action
}

func (e SleepEvent) webhookEvent()                     {}
func (e SleepEvent) GetAction() storage.Action         { return e.action }
func (e SleepEvent) GetEntityType() storage.EntityType { return storage.EntityTypeSleep }
func (e SleepEvent) GetEntityID() string               { return e.SleepID }

type RecoveryEvent struct {
	eventBase
	SleepID string `json:"id"`
	action  storage.Action
}

func (e RecoveryEvent) webhookEvent()                     {}
func (e RecoveryEvent) GetAction() storage.Action         { return e.action }
func (e RecoveryEvent) GetEntityType() storage.EntityType { return storage.EntityTypeRecovery }
func (e RecoveryEvent) GetEntityID() string               { return e.SleepID }

type rawPayload struct {
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
	ID     string `json:"id"`
}

// ParseEvent parses a raw webhook payload into a typed event.
// Returns ErrUnknownEventType for unknown entity types or actions.
func ParseEvent(data []byte) (Event, error) {
	var raw rawPayload
	if err := go_json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	parts := strings.SplitN(raw.Type, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: invalid event type format: %s", ErrUnknownEventType, raw.Type)
	}

	entity, actionStr := parts[0], parts[1]

	var action storage.Action
	switch actionStr {
	case "updated":
		action = storage.ActionUpdated
	case "deleted":
		action = storage.ActionDeleted
	default:
		return nil, fmt.Errorf("%w: unknown action: %s", ErrUnknownEventType, actionStr)
	}

	base := eventBase{Type: raw.Type, UserID: raw.UserID}

	switch entity {
	case "workout":
		return WorkoutEvent{eventBase: base, WorkoutID: raw.ID, action: action}, nil
	case "sleep":
		return SleepEvent{eventBase: base, SleepID: raw.ID, action: action}, nil
	case "recovery":
		return RecoveryEvent{eventBase: base, SleepID: raw.ID, action: action}, nil
	default:
		return nil, fmt.Errorf("%w: unknown entity type: %s", ErrUnknownEventType, entity)
	}
}
