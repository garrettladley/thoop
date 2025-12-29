package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/garrettladley/thoop/internal/sqlc"
)

type syncStateRepo struct {
	q sqlc.Querier
}

func (r *syncStateRepo) Get(ctx context.Context) (*SyncState, error) {
	row, err := r.q.GetSyncState(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		if err := r.q.UpsertSyncState(ctx, sqlc.UpsertSyncStateParams{}); err != nil {
			return nil, err
		}
		return &SyncState{}, nil
	}
	if err != nil {
		return nil, err
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

	return r.q.UpsertSyncState(ctx, sqlc.UpsertSyncStateParams{
		BackfillComplete:  backfillComplete,
		BackfillWatermark: state.BackfillWatermark,
		LastFullSync:      state.LastFullSync,
	})
}

func (r *syncStateRepo) MarkBackfillComplete(ctx context.Context) error {
	return r.q.MarkBackfillComplete(ctx)
}

func (r *syncStateRepo) UpdateBackfillWatermark(ctx context.Context, watermark time.Time) error {
	return r.q.UpdateBackfillWatermark(ctx, &watermark)
}

func (r *syncStateRepo) UpdateLastFullSync(ctx context.Context, syncTime time.Time) error {
	return r.q.UpdateLastFullSync(ctx, &syncTime)
}

func (r *syncStateRepo) GetLastNotificationPoll(ctx context.Context) (*time.Time, error) {
	return r.q.GetLastNotificationPoll(ctx)
}

func (r *syncStateRepo) UpdateLastNotificationPoll(ctx context.Context, pollTime time.Time) error {
	return r.q.UpdateLastNotificationPoll(ctx, &pollTime)
}
