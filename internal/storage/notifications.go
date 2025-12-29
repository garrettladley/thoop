package storage

import (
	"context"
	"fmt"
	"time"

	go_json "github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

const (
	notificationsKeyPrefix  = "notifications:"
	notificationsLivePrefix = "notifications:live:"
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
	EntityType EntityType `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	Action     Action     `json:"action"`
	Timestamp  time.Time  `json:"timestamp"`
}

type NotificationStore interface {
	Add(ctx context.Context, userID string, n Notification) error

	GetSince(ctx context.Context, userID string, since time.Time) ([]Notification, error)

	Publish(ctx context.Context, userID string, n Notification) error

	// Subscribe returns a channel that receives notifications for a user.
	// The returned function should be called to unsubscribe.
	Subscribe(ctx context.Context, userID string) (<-chan Notification, func(), error)

	// DeleteBefore removes notifications older than the given timestamp.
	// Called after client successfully processes notifications.
	DeleteBefore(ctx context.Context, userID string, before time.Time) error
}

var _ NotificationStore = (*RedisNotificationStore)(nil)

type RedisNotificationStore struct {
	client *redis.Client
}

func NewRedisNotificationStore(client *redis.Client) *RedisNotificationStore {
	return &RedisNotificationStore{
		client: client,
	}
}

func (s *RedisNotificationStore) notificationsKey(userID string) string {
	return notificationsKeyPrefix + userID
}

func (s *RedisNotificationStore) liveKey(userID string) string {
	return notificationsLivePrefix + userID
}

func (s *RedisNotificationStore) Add(ctx context.Context, userID string, n Notification) error {
	data, err := go_json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	key := s.notificationsKey(userID)
	score := float64(n.Timestamp.UnixMilli())

	err = s.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add notification: %w", err)
	}

	return nil
}

func (s *RedisNotificationStore) GetSince(ctx context.Context, userID string, since time.Time) ([]Notification, error) {
	key := s.notificationsKey(userID)
	minScore := fmt.Sprintf("%d", since.UnixMilli())

	results, err := s.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: "(" + minScore,
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	notifications := make([]Notification, 0, len(results))
	for _, data := range results {
		var n Notification
		if err := go_json.Unmarshal([]byte(data), &n); err != nil {
			continue
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

func (s *RedisNotificationStore) Publish(ctx context.Context, userID string, n Notification) error {
	data, err := go_json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	channel := s.liveKey(userID)
	if err := s.client.Publish(ctx, channel, string(data)).Err(); err != nil {
		return fmt.Errorf("failed to publish notification: %w", err)
	}

	return nil
}

func (s *RedisNotificationStore) Subscribe(ctx context.Context, userID string) (<-chan Notification, func(), error) {
	channel := s.liveKey(userID)
	pubsub := s.client.Subscribe(ctx, channel)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		_ = pubsub.Close()
		return nil, nil, fmt.Errorf("failed to subscribe: %w", err)
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

func (s *RedisNotificationStore) DeleteBefore(ctx context.Context, userID string, before time.Time) error {
	key := s.notificationsKey(userID)
	maxScore := fmt.Sprintf("%d", before.UnixMilli())

	err := s.client.ZRemRangeByScore(ctx, key, "-inf", maxScore).Err()
	if err != nil {
		return fmt.Errorf("failed to delete notifications: %w", err)
	}

	return nil
}
