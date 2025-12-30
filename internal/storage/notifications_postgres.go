package storage

import (
	"context"
	"fmt"
	"strconv"

	go_json "github.com/goccy/go-json"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	pgc "github.com/garrettladley/thoop/internal/sqlc/postgres"
)

const notificationsLivePrefix = "notifications:live:"

var _ NotificationStore = (*HybridNotificationStore)(nil)

type HybridNotificationStore struct {
	queries *pgc.Queries
	redis   *redis.Client
}

func NewHybridNotificationStore(pool *pgxpool.Pool, redis *redis.Client) *HybridNotificationStore {
	return &HybridNotificationStore{
		queries: pgc.New(pool),
		redis:   redis,
	}
}

func (s *HybridNotificationStore) liveKey(userID int64) string {
	return notificationsLivePrefix + strconv.FormatInt(userID, 10)
}

func (s *HybridNotificationStore) Add(ctx context.Context, userID int64, n Notification) error {
	id, err := s.queries.InsertWebhookEvent(ctx, pgc.InsertWebhookEventParams{
		TraceID:     n.TraceID,
		WhoopUserID: userID,
		Timestamp:   pgtype.Timestamptz{Time: n.Timestamp, Valid: true},
		EntityID:    n.EntityID,
		EntityType:  string(n.EntityType),
		Action:      string(n.Action),
	})
	if err != nil {
		return fmt.Errorf("insert webhook event: %w", err)
	}

	// id is nil when ON CONFLICT DO NOTHING triggers (duplicate trace_id)
	// In that case, skip publishing since it was already published
	if id == nil {
		return nil
	}

	// Set the ID on the notification for real-time subscribers
	n.ID = *id

	data, err := go_json.Marshal(n)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	if err := s.redis.Publish(ctx, s.liveKey(userID), string(data)).Err(); err != nil {
		return fmt.Errorf("publish notification: %w", err)
	}

	return nil
}

func (s *HybridNotificationStore) GetUnacked(ctx context.Context, userID int64, cursor int64, limit int32) ([]Notification, error) {
	events, err := s.queries.GetUnackedWebhookEvents(ctx, pgc.GetUnackedWebhookEventsParams{
		WhoopUserID: userID,
		Cursor:      &cursor,
		MaxResults:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get unacked webhook events: %w", err)
	}

	notifications := make([]Notification, 0, len(events))
	for _, e := range events {
		var id int64
		if e.ID != nil {
			id = *e.ID
		}
		notifications = append(notifications, Notification{
			ID:         id,
			TraceID:    e.TraceID,
			EntityType: EntityType(e.EntityType),
			EntityID:   e.EntityID,
			Action:     Action(e.Action),
			Timestamp:  e.Timestamp.Time,
		})
	}
	return notifications, nil
}

func (s *HybridNotificationStore) Acknowledge(ctx context.Context, userID int64, traceIDs []string) error {
	err := s.queries.AcknowledgeWebhookEventsByTraceIDs(ctx, pgc.AcknowledgeWebhookEventsByTraceIDsParams{
		WhoopUserID: userID,
		TraceIds:    traceIDs,
	})
	if err != nil {
		return fmt.Errorf("acknowledge webhook events: %w", err)
	}
	return nil
}

func (s *HybridNotificationStore) Subscribe(ctx context.Context, userID int64) (<-chan Notification, func(), error) {
	channel := s.liveKey(userID)
	pubsub := s.redis.Subscribe(ctx, channel)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		_ = pubsub.Close()
		return nil, nil, fmt.Errorf("subscribe: %w", err)
	}

	notifCh := make(chan Notification)

	go func() {
		defer close(notifCh)
		ch := pubsub.Channel()

		for msg := range ch {
			var n Notification
			if err := go_json.Unmarshal([]byte(msg.Payload), &n); err != nil {
				continue
			}

			select {
			case notifCh <- n:
			case <-ctx.Done():
				return
			}
		}
	}()

	unsubscribe := func() {
		_ = pubsub.Close()
	}

	return notifCh, unsubscribe, nil
}
