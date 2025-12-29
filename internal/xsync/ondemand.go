package xsync

import (
	"context"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xslog"
)

// fetchHistorical fetches data for a specific date range.
// Used when the user expands the time horizon in the TUI.
func (s *Service) fetchHistorical(ctx context.Context, start, end time.Time) error {
	s.logger.InfoContext(ctx, "fetching historical data",
		xslog.Start(start),
		xslog.End(end))

	// Fetch cycles
	if err := s.fetchHistoricalCycles(ctx, start, end); err != nil {
		return err
	}

	// Fetch sleeps
	if err := s.fetchHistoricalSleeps(ctx, start, end); err != nil {
		return err
	}

	// Fetch workouts
	if err := s.fetchHistoricalWorkouts(ctx, start, end); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "historical data fetch complete")
	return nil
}

// fetchHistoricalCycles fetches cycles for a date range with pagination.
func (s *Service) fetchHistoricalCycles(ctx context.Context, start, end time.Time) error {
	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: BackfillPageSize,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := s.client.Cycle.List(ctx, params)
		if err != nil {
			return err
		}

		if err := s.repo.Cycles.UpsertBatch(ctx, resp.Records); err != nil {
			return err
		}

		// Fetch recoveries for each cycle
		for _, cycle := range resp.Records {
			recovery, err := s.client.Cycle.GetRecovery(ctx, cycle.ID)
			if err != nil {
				s.logger.WarnContext(ctx, "failed to fetch recovery",
					xslog.CycleID(cycle.ID),
					xslog.Error(err))
				continue
			}
			if err := s.repo.Recoveries.Upsert(ctx, recovery); err != nil {
				s.logger.WarnContext(ctx, "failed to save recovery",
					xslog.CycleID(cycle.ID),
					xslog.Error(err))
			}
		}

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	return nil
}

// fetchHistoricalSleeps fetches sleeps for a date range with pagination.
func (s *Service) fetchHistoricalSleeps(ctx context.Context, start, end time.Time) error {
	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: BackfillPageSize,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := s.client.Sleep.List(ctx, params)
		if err != nil {
			return err
		}

		if err := s.repo.Sleeps.UpsertBatch(ctx, resp.Records); err != nil {
			return err
		}

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	return nil
}

// fetchHistoricalWorkouts fetches workouts for a date range with pagination.
func (s *Service) fetchHistoricalWorkouts(ctx context.Context, start, end time.Time) error {
	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: BackfillPageSize,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := s.client.Workout.List(ctx, params)
		if err != nil {
			return err
		}

		if err := s.repo.Workouts.UpsertBatch(ctx, resp.Records); err != nil {
			return err
		}

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	return nil
}
