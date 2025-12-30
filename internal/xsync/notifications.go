package xsync

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
)

type NotificationProcessor struct {
	client *whoop.Client
	repo   *repository.Repository
	logger *slog.Logger
}

func NewNotificationProcessor(client *whoop.Client, repo *repository.Repository, logger *slog.Logger) *NotificationProcessor {
	return &NotificationProcessor{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

type ProcessResult struct {
	EntityType storage.EntityType
	Action     storage.Action
	EntityID   string
	Success    bool
}

func (p *NotificationProcessor) Process(ctx context.Context, n storage.Notification) ProcessResult {
	result := ProcessResult{
		EntityType: n.EntityType,
		Action:     n.Action,
		EntityID:   n.EntityID,
	}

	var err error
	switch n.Action {
	case storage.ActionUpdated:
		err = p.handleUpdate(ctx, n)
	case storage.ActionDeleted:
		err = p.handleDelete(ctx, n)
	default:
		p.logger.WarnContext(ctx, "unknown notification action",
			xslog.Action(string(n.Action)),
			xslog.EntityType(string(n.EntityType)),
		)
		return result
	}

	if err != nil {
		p.logger.ErrorContext(ctx, "failed to process notification",
			xslog.Error(err),
			xslog.EntityType(string(n.EntityType)),
			xslog.Action(string(n.Action)),
			xslog.EntityID(n.EntityID),
		)
		return result
	}

	result.Success = true
	p.logger.InfoContext(ctx, "processed notification",
		xslog.EntityType(string(n.EntityType)),
		xslog.Action(string(n.Action)),
		xslog.EntityID(n.EntityID),
	)
	return result
}

func (p *NotificationProcessor) handleUpdate(ctx context.Context, n storage.Notification) error {
	switch n.EntityType {
	case storage.EntityTypeWorkout:
		return p.fetchAndCacheWorkout(ctx, n.EntityID)
	case storage.EntityTypeSleep:
		return p.fetchAndCacheSleep(ctx, n.EntityID)
	case storage.EntityTypeRecovery:
		return p.fetchAndCacheRecoveryBySleepID(ctx, n.EntityID)
	default:
		return nil
	}
}

func (p *NotificationProcessor) handleDelete(ctx context.Context, n storage.Notification) error {
	switch n.EntityType {
	case storage.EntityTypeWorkout:
		if err := p.repo.Workouts.Delete(ctx, n.EntityID); err != nil {
			return fmt.Errorf("failed to delete workout: %w", err)
		}
		return nil
	case storage.EntityTypeSleep:
		if err := p.repo.Sleeps.Delete(ctx, n.EntityID); err != nil {
			return fmt.Errorf("failed to delete sleep: %w", err)
		}
		return nil
	case storage.EntityTypeRecovery:
		sleep, err := p.repo.Sleeps.Get(ctx, n.EntityID)
		if err != nil {
			return fmt.Errorf("failed to get sleep: %w", err)
		}
		if sleep != nil {
			if err := p.repo.Recoveries.Delete(ctx, sleep.CycleID); err != nil {
				return fmt.Errorf("failed to delete recovery: %w", err)
			}
		}
		return nil
	default:
		return nil
	}
}

func (p *NotificationProcessor) fetchAndCacheWorkout(ctx context.Context, id string) error {
	workout, err := p.client.Workout.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workout: %w", err)
	}
	if err := p.repo.Workouts.Upsert(ctx, workout); err != nil {
		return fmt.Errorf("failed to upsert workout: %w", err)
	}
	return nil
}

func (p *NotificationProcessor) fetchAndCacheSleep(ctx context.Context, id string) error {
	sleep, err := p.client.Sleep.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get sleep: %w", err)
	}
	if err := p.repo.Sleeps.Upsert(ctx, sleep); err != nil {
		return fmt.Errorf("failed to upsert sleep: %w", err)
	}
	return nil
}

func (p *NotificationProcessor) fetchAndCacheRecoveryBySleepID(ctx context.Context, sleepID string) error {
	sleep, err := p.repo.Sleeps.Get(ctx, sleepID)
	if err != nil {
		return fmt.Errorf("failed to get sleep: %w", err)
	}
	if sleep == nil {
		sleep, err = p.client.Sleep.Get(ctx, sleepID)
		if err != nil {
			return fmt.Errorf("failed to get sleep from api: %w", err)
		}
		if err := p.repo.Sleeps.Upsert(ctx, sleep); err != nil {
			p.logger.WarnContext(ctx, "failed to cache sleep", xslog.Error(err))
		}
	}

	recovery, err := p.client.Cycle.GetRecovery(ctx, sleep.CycleID)
	if err != nil {
		return fmt.Errorf("failed to get recovery: %w", err)
	}
	if err := p.repo.Recoveries.Upsert(ctx, recovery); err != nil {
		return fmt.Errorf("failed to upsert recovery: %w", err)
	}
	return nil
}
