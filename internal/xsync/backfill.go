package xsync

import (
	"context"
	"fmt"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/sync/errgroup"
)

const (
	BackfillDuration = 30 * 24 * time.Hour

	BackfillPageSize = 10

	maxRecoveryConcurrency = 2
)

func (s *Service) runBackfill(ctx context.Context) {
	s.logger.InfoContext(ctx, "starting backfill")

	end := time.Now()
	start := end.Add(-BackfillDuration)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.backfillCycles(gctx, start, end) })
	g.Go(func() error { return s.backfillSleeps(gctx, start, end) })
	g.Go(func() error { return s.backfillWorkouts(gctx, start, end) })
	if err := g.Wait(); err != nil {
		s.logger.ErrorContext(ctx, "backfill failed", xslog.Error(err))
		return
	}

	if err := s.repo.SyncState.MarkBackfillComplete(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to mark backfill complete", xslog.Error(err))
		return
	}

	s.logger.InfoContext(ctx, "backfill complete")
}

func (s *Service) backfillCycles(ctx context.Context, start, end time.Time) error {
	s.logger.InfoContext(ctx, "backfilling cycles",
		xslog.Start(start),
		xslog.End(end))

	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: BackfillPageSize,
	}

	totalCycles := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		resp, err := s.client.Cycle.List(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to list cycles: %w", err)
		}

		if err := s.repo.Cycles.UpsertBatch(ctx, resp.Records); err != nil {
			return fmt.Errorf("failed to upsert cycles batch: %w", err)
		}

		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(maxRecoveryConcurrency)
		for _, cycle := range resp.Records {
			g.Go(func() error {
				if err := s.backfillRecoveryForCycle(gctx, cycle.ID); err != nil {
					s.logger.WarnContext(
						gctx,
						"failed to backfill recovery",
						xslog.CycleID(cycle.ID),
						xslog.Error(err),
					)
				}
				return nil
			})
		}
		_ = g.Wait()

		totalCycles += len(resp.Records)

		if len(resp.Records) > 0 {
			oldest := resp.Records[len(resp.Records)-1].Start
			if err := s.repo.SyncState.UpdateBackfillWatermark(ctx, oldest); err != nil {
				s.logger.WarnContext(ctx, "failed to update backfill watermark", xslog.Error(err))
			}
		}

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	s.logger.InfoContext(ctx, "backfilled cycles", xslog.Count(totalCycles))
	return nil
}

func (s *Service) backfillRecoveryForCycle(ctx context.Context, cycleID int64) error {
	recovery, err := s.client.Cycle.GetRecovery(ctx, cycleID)
	if err != nil {
		return fmt.Errorf("failed to get recovery: %w", err)
	}
	if err := s.repo.Recoveries.Upsert(ctx, recovery); err != nil {
		return fmt.Errorf("failed to upsert recovery: %w", err)
	}
	return nil
}

func (s *Service) backfillSleeps(ctx context.Context, start, end time.Time) error {
	s.logger.InfoContext(ctx, "backfilling sleeps",
		xslog.Start(start),
		xslog.End(end))

	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: BackfillPageSize,
	}

	totalSleeps := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		resp, err := s.client.Sleep.List(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to list sleeps: %w", err)
		}

		if err := s.repo.Sleeps.UpsertBatch(ctx, resp.Records); err != nil {
			return fmt.Errorf("failed to upsert sleeps batch: %w", err)
		}

		totalSleeps += len(resp.Records)

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	s.logger.InfoContext(ctx, "backfilled sleeps", xslog.Count(totalSleeps))
	return nil
}

func (s *Service) backfillWorkouts(ctx context.Context, start, end time.Time) error {
	s.logger.InfoContext(ctx, "backfilling workouts",
		xslog.Start(start),
		xslog.End(end))

	params := &whoop.ListParams{
		Start: &start,
		End:   &end,
		Limit: BackfillPageSize,
	}

	totalWorkouts := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		resp, err := s.client.Workout.List(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to list workouts: %w", err)
		}

		if err := s.repo.Workouts.UpsertBatch(ctx, resp.Records); err != nil {
			return fmt.Errorf("failed to upsert workouts batch: %w", err)
		}

		totalWorkouts += len(resp.Records)

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	s.logger.InfoContext(ctx, "backfilled workouts", xslog.Count(totalWorkouts))
	return nil
}
