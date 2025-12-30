package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
)

type syncStateRepo struct {
	q sqlitec.Querier
}

func (r *syncStateRepo) Get(ctx context.Context) (*SyncState, error) {
	row, err := r.q.GetSyncState(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		if err := r.q.UpsertSyncState(ctx, sqlitec.UpsertSyncStateParams{}); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		return &SyncState{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return &SyncState{
		BackfillComplete:     row.BackfillComplete == 1,
		BackfillWatermark:    row.BackfillWatermark,
		LastFullSync:         row.LastFullSync,
		LastNotificationPoll: row.LastNotificationPoll,
	}, nil
}

func (r *syncStateRepo) Upsert(ctx context.Context, state *SyncState) error {
	var backfillComplete int64
	if state.BackfillComplete {
		backfillComplete = 1
	}

	err := r.q.UpsertSyncState(ctx, sqlitec.UpsertSyncStateParams{
		BackfillComplete:  backfillComplete,
		BackfillWatermark: state.BackfillWatermark,
		LastFullSync:      state.LastFullSync,
	})
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (r *syncStateRepo) MarkBackfillComplete(ctx context.Context) error {
	err := r.q.MarkBackfillComplete(ctx)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (r *syncStateRepo) UpdateBackfillWatermark(ctx context.Context, watermark time.Time) error {
	err := r.q.UpdateBackfillWatermark(ctx, &watermark)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (r *syncStateRepo) UpdateLastFullSync(ctx context.Context, syncTime time.Time) error {
	err := r.q.UpdateLastFullSync(ctx, &syncTime)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (r *syncStateRepo) GetLastNotificationPoll(ctx context.Context) (*time.Time, error) {
	result, err := r.q.GetLastNotificationPoll(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return result, nil
}

func (r *syncStateRepo) UpdateLastNotificationPoll(ctx context.Context, pollTime time.Time) error {
	err := r.q.UpdateLastNotificationPoll(ctx, &pollTime)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
