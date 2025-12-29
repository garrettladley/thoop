package xsync

import (
	"context"
	"log/slog"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/xslog"
)

// DataFetcher provides cache-aware access to WHOOP data.
// It checks the local cache first and falls back to the API when needed.
type DataFetcher interface {
	// GetCurrentCycle returns the current (most recent) cycle.
	// Always fetches from API since current cycle may be PENDING_SCORE.
	GetCurrentCycle(ctx context.Context) (*whoop.Cycle, error)

	GetCycles(ctx context.Context, start, end time.Time) ([]whoop.Cycle, error)

	GetRecovery(ctx context.Context, cycleID int64) (*whoop.Recovery, error)

	GetSleep(ctx context.Context, cycleID int64) (*whoop.Sleep, error)

	GetWorkouts(ctx context.Context, start, end time.Time) ([]whoop.Workout, error)
}

type Fetcher struct {
	client *whoop.Client
	repo   *repository.Repository
	logger *slog.Logger
}

var _ DataFetcher = (*Fetcher)(nil)

func NewFetcher(client *whoop.Client, repo *repository.Repository, logger *slog.Logger) *Fetcher {
	return &Fetcher{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

func (f *Fetcher) GetCurrentCycle(ctx context.Context) (*whoop.Cycle, error) {
	resp, err := f.client.Cycle.List(ctx, &whoop.ListParams{Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(resp.Records) == 0 {
		return nil, nil
	}

	cycle := &resp.Records[0]

	if err := f.repo.Cycles.Upsert(ctx, cycle); err != nil {
		f.logger.WarnContext(ctx, "failed to cache current cycle", xslog.Error(err))
	}

	return cycle, nil
}

func (f *Fetcher) GetCycles(ctx context.Context, start, end time.Time) ([]whoop.Cycle, error) {
	result, err := f.repo.Cycles.GetByDateRange(ctx, start, end, nil)
	if err != nil {
		return nil, err
	}

	if len(result.Records) > 0 {
		return result.Records, nil
	}

	return f.fetchCyclesFromAPI(ctx, start, end)
}

func (f *Fetcher) GetRecovery(ctx context.Context, cycleID int64) (*whoop.Recovery, error) {
	recovery, err := f.repo.Recoveries.Get(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	if recovery != nil {
		return recovery, nil
	}

	recovery, err = f.client.Cycle.GetRecovery(ctx, cycleID)
	if err != nil {
		return nil, err
	}

	if err := f.repo.Recoveries.Upsert(ctx, recovery); err != nil {
		f.logger.WarnContext(ctx, "failed to cache recovery", xslog.Error(err))
	}

	return recovery, nil
}

func (f *Fetcher) GetSleep(ctx context.Context, cycleID int64) (*whoop.Sleep, error) {
	sleep, err := f.repo.Sleeps.GetByCycleID(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	if sleep != nil {
		return sleep, nil
	}

	sleep, err = f.client.Cycle.GetSleep(ctx, cycleID)
	if err != nil {
		return nil, err
	}

	if err := f.repo.Sleeps.Upsert(ctx, sleep); err != nil {
		f.logger.WarnContext(ctx, "failed to cache sleep", xslog.Error(err))
	}

	return sleep, nil
}

func (f *Fetcher) GetWorkouts(ctx context.Context, start, end time.Time) ([]whoop.Workout, error) {
	result, err := f.repo.Workouts.GetByDateRange(ctx, start, end, nil)
	if err != nil {
		return nil, err
	}

	if len(result.Records) > 0 {
		return result.Records, nil
	}

	return f.fetchWorkoutsFromAPI(ctx, start, end)
}

func (f *Fetcher) fetchCyclesFromAPI(ctx context.Context, start, end time.Time) ([]whoop.Cycle, error) {
	var allCycles []whoop.Cycle
	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: 25,
	}

	for {
		resp, err := f.client.Cycle.List(ctx, params)
		if err != nil {
			return nil, err
		}

		if err := f.repo.Cycles.UpsertBatch(ctx, resp.Records); err != nil {
			f.logger.WarnContext(ctx, "failed to cache cycles batch", xslog.Error(err))
		}

		allCycles = append(allCycles, resp.Records...)

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	return allCycles, nil
}

func (f *Fetcher) fetchWorkoutsFromAPI(ctx context.Context, start, end time.Time) ([]whoop.Workout, error) {
	var allWorkouts []whoop.Workout
	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: 25,
	}

	for {
		resp, err := f.client.Workout.List(ctx, params)
		if err != nil {
			return nil, err
		}

		if err := f.repo.Workouts.UpsertBatch(ctx, resp.Records); err != nil {
			f.logger.WarnContext(ctx, "failed to cache workouts batch", xslog.Error(err))
		}

		allWorkouts = append(allWorkouts, resp.Records...)

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	return allWorkouts, nil
}
