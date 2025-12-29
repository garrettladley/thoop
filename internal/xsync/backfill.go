package xsync

import (
	"context"
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

	if err := s.backfillCycles(ctx, start, end); err != nil {
		s.logger.ErrorContext(ctx, "backfill cycles failed", xslog.Error(err))
		return
	}

	if err := s.backfillSleeps(ctx, start, end); err != nil {
		s.logger.ErrorContext(ctx, "backfill sleeps failed", xslog.Error(err))
		return
	}

	if err := s.backfillWorkouts(ctx, start, end); err != nil {
		s.logger.ErrorContext(ctx, "backfill workouts failed", xslog.Error(err))
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

		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(maxRecoveryConcurrency)
		for _, cycle := range resp.Records {
			g.Go(func() error {
				if err := s.backfillRecoveryForCycle(gctx, cycle.ID); err != nil {
					s.logger.WarnContext(gctx, "failed to backfill recovery",
						xslog.CycleID(cycle.ID), xslog.Error(err))
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
		return err
	}
	return s.repo.Recoveries.Upsert(ctx, recovery)
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

		totalWorkouts += len(resp.Records)

		if !resp.HasMore() {
			break
		}
		params.NextToken = resp.NextToken
	}

	s.logger.InfoContext(ctx, "backfilled workouts", xslog.Count(totalWorkouts))
	return nil
}
