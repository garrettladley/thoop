package notification

import (
	"context"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
)

type Store struct {
	store storage.NotificationStore
}

var _ Service = (*Store)(nil)

func NewStore(store storage.NotificationStore) *Store {
	return &Store{store: store}
}

func (s *Store) GetNotifications(ctx context.Context, userID int64, since time.Time) (*PollResult, error) {
	notifications, err := s.store.GetSince(ctx, userID, since)
	if err != nil {
		return nil, err
	}

	if notifications == nil {
		notifications = []storage.Notification{}
	}

	return &PollResult{
		Notifications: notifications,
		ServerTime:    time.Now(),
	}, nil
}

func (s *Store) Subscribe(ctx context.Context, userID int64) (<-chan storage.Notification, func(), error) {
	return s.store.Subscribe(ctx, userID)
}
