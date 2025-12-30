package xsync

import (
	"context"
	"fmt"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/sync/errgroup"
)

// fetchHistorical fetches data for a specific date range.
// Used when the user expands the time horizon in the TUI.
func (s *Service) fetchHistorical(ctx context.Context, start, end time.Time) error {
	s.logger.InfoContext(ctx, "fetching historical data",
		xslog.Start(start),
		xslog.End(end))

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.fetchHistoricalCycles(gctx, start, end) })
	g.Go(func() error { return s.fetchHistoricalSleeps(gctx, start, end) })
	g.Go(func() error { return s.fetchHistoricalWorkouts(gctx, start, end) })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("%w", err)
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
			return fmt.Errorf("%w", ctx.Err())
		default:
		}

		resp, err := s.client.Cycle.List(ctx, params)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		if err := s.repo.Cycles.UpsertBatch(ctx, resp.Records); err != nil {
			return fmt.Errorf("%w", err)
		}

		// Fetch recoveries for each cycle in parallel
		rg, rgctx := errgroup.WithContext(ctx)
		rg.SetLimit(maxRecoveryConcurrency)
		for _, cycle := range resp.Records {
			rg.Go(func() error {
				recovery, err := s.client.Cycle.GetRecovery(rgctx, cycle.ID)
				if err != nil {
					s.logger.WarnContext(rgctx, "failed to fetch recovery",
						xslog.CycleID(cycle.ID),
						xslog.Error(err))
					return nil
				}
				if err := s.repo.Recoveries.Upsert(rgctx, recovery); err != nil {
					s.logger.WarnContext(rgctx, "failed to save recovery",
						xslog.CycleID(cycle.ID),
						xslog.Error(err))
				}
				return nil
			})
		}
		_ = rg.Wait()

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
			return fmt.Errorf("%w", ctx.Err())
		default:
		}

		resp, err := s.client.Sleep.List(ctx, params)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		if err := s.repo.Sleeps.UpsertBatch(ctx, resp.Records); err != nil {
			return fmt.Errorf("%w", err)
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
			return fmt.Errorf("%w", ctx.Err())
		default:
		}

		resp, err := s.client.Workout.List(ctx, params)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		if err := s.repo.Workouts.UpsertBatch(ctx, resp.Records); err != nil {
			return fmt.Errorf("%w", err)
		}

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	return nil
}
