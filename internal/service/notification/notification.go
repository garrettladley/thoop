package notification

import (
	"context"
	"fmt"

	"github.com/garrettladley/thoop/internal/storage"
)

type Store struct {
	store storage.NotificationStore
}

var _ Service = (*Store)(nil)

func NewStore(store storage.NotificationStore) *Store {
	return &Store{store: store}
}

func (s *Store) GetUnacked(ctx context.Context, userID int64, cursor int64, limit int32) (*PollResult, error) {
	notifications, err := s.store.GetUnacked(ctx, userID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("getting unacked notifications: %w", err)
	}

	if notifications == nil {
		notifications = []storage.Notification{}
	}

	return &PollResult{
		Notifications: notifications,
	}, nil
}

func (s *Store) Acknowledge(ctx context.Context, userID int64, traceIDs []string) error {
	if err := s.store.Acknowledge(ctx, userID, traceIDs); err != nil {
		return fmt.Errorf("acknowledging notifications: %w", err)
	}
	return nil
}

func (s *Store) Subscribe(ctx context.Context, userID int64) (<-chan storage.Notification, func(), error) {
	ch, cleanup, err := s.store.Subscribe(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("subscribing to notifications: %w", err)
	}
	return ch, cleanup, nil
}
