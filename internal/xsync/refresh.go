package xsync

import (
	"context"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/sync/errgroup"
)

// refreshCurrent fetches the current cycle and the previous (n-1) cycle.
// This is called on TUI startup to ensure fresh data for display.
func (s *Service) refreshCurrent(ctx context.Context) error {
	s.logger.InfoContext(ctx, "refreshing current cycles")

	// fetch the 2 most recent cycles
	resp, err := s.client.Cycle.List(ctx, &whoop.ListParams{Limit: 2})
	if err != nil {
		return err
	}

	if len(resp.Records) == 0 {
		s.logger.InfoContext(ctx, "no cycles found")
		return nil
	}

	// save cycles
	if err := s.repo.Cycles.UpsertBatch(ctx, resp.Records); err != nil {
		return err
	}

	// fetch recovery and sleep for each cycle in parallel
	g, gctx := errgroup.WithContext(ctx)
	for _, cycle := range resp.Records {
		g.Go(func() error {
			if err := s.refreshRecoveryForCycle(gctx, cycle.ID); err != nil {
				s.logger.WarnContext(gctx, "failed to refresh recovery",
					xslog.CycleID(cycle.ID),
					xslog.Error(err))
			}
			return nil
		})
		g.Go(func() error {
			if err := s.refreshSleepForCycle(gctx, cycle.ID); err != nil {
				s.logger.WarnContext(gctx, "failed to refresh sleep",
					xslog.CycleID(cycle.ID),
					xslog.Error(err))
			}
			return nil
		})
	}
	_ = g.Wait()

	// update last sync time
	now := time.Now()
	if err := s.repo.SyncState.UpdateLastFullSync(ctx, now); err != nil {
		s.logger.WarnContext(ctx, "failed to update last sync time", xslog.Error(err))
	}

	s.logger.InfoContext(ctx, "refreshed current cycles", xslog.Count(len(resp.Records)))
	return nil
}

func (s *Service) refreshRecoveryForCycle(ctx context.Context, cycleID int64) error {
	recovery, err := s.client.Cycle.GetRecovery(ctx, cycleID)
	if err != nil {
		return err
	}
	return s.repo.Recoveries.Upsert(ctx, recovery)
}

func (s *Service) refreshSleepForCycle(ctx context.Context, cycleID int64) error {
	sleep, err := s.client.Cycle.GetSleep(ctx, cycleID)
	if err != nil {
		return err
	}
	return s.repo.Sleeps.Upsert(ctx, sleep)
}
