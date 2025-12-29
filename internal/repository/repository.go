package repository

import (
	"context"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/sqlc"
)

type Repository struct {
	SyncState  SyncStateRepository
	Cycles     CycleRepository
	Recoveries RecoveryRepository
	Sleeps     SleepRepository
	Workouts   WorkoutRepository
}

func New(q sqlc.Querier) *Repository {
	return &Repository{
		SyncState:  &syncStateRepo{q: q},
		Cycles:     &cycleRepo{q: q},
		Recoveries: &recoveryRepo{q: q},
		Sleeps:     &sleepRepo{q: q},
		Workouts:   &workoutRepo{q: q},
	}
}

type CursorParams struct {
	Limit  int
	Cursor *time.Time
}

type CursorResult[T any] struct {
	Records    []T
	NextCursor *time.Time
}

const DefaultPageSize = 50

type SyncState struct {
	BackfillComplete     bool
	BackfillWatermark    *time.Time
	LastFullSync         *time.Time
	LastNotificationPoll *time.Time
}

type SyncStateRepository interface {
	Get(ctx context.Context) (*SyncState, error)
	Upsert(ctx context.Context, state *SyncState) error
	MarkBackfillComplete(ctx context.Context) error
	UpdateBackfillWatermark(ctx context.Context, watermark time.Time) error
	UpdateLastFullSync(ctx context.Context, syncTime time.Time) error
	GetLastNotificationPoll(ctx context.Context) (*time.Time, error)
	UpdateLastNotificationPoll(ctx context.Context, pollTime time.Time) error
}

type CycleRepository interface {
	Upsert(ctx context.Context, cycle *whoop.Cycle) error
	UpsertBatch(ctx context.Context, cycles []whoop.Cycle) error
	Get(ctx context.Context, id int64) (*whoop.Cycle, error)
	GetLatest(ctx context.Context, limit int) ([]whoop.Cycle, error)
	GetByDateRange(ctx context.Context, start, end time.Time, cursor *CursorParams) (*CursorResult[whoop.Cycle], error)
	GetPending(ctx context.Context) ([]whoop.Cycle, error)
}

type RecoveryRepository interface {
	Upsert(ctx context.Context, recovery *whoop.Recovery) error
	UpsertBatch(ctx context.Context, recoveries []whoop.Recovery) error
	Get(ctx context.Context, cycleID int64) (*whoop.Recovery, error)
	GetByCycleIDs(ctx context.Context, cycleIDs []int64) ([]whoop.Recovery, error)
	Delete(ctx context.Context, cycleID int64) error
}

type SleepRepository interface {
	Upsert(ctx context.Context, sleep *whoop.Sleep) error
	UpsertBatch(ctx context.Context, sleeps []whoop.Sleep) error
	Get(ctx context.Context, id string) (*whoop.Sleep, error)
	GetByCycleID(ctx context.Context, cycleID int64) (*whoop.Sleep, error)
	GetByDateRange(ctx context.Context, start, end time.Time, cursor *CursorParams) (*CursorResult[whoop.Sleep], error)
	Delete(ctx context.Context, id string) error
}

type WorkoutRepository interface {
	Upsert(ctx context.Context, workout *whoop.Workout) error
	UpsertBatch(ctx context.Context, workouts []whoop.Workout) error
	Get(ctx context.Context, id string) (*whoop.Workout, error)
	GetByDateRange(ctx context.Context, start, end time.Time, cursor *CursorParams) (*CursorResult[whoop.Workout], error)
	Delete(ctx context.Context, id string) error
}
