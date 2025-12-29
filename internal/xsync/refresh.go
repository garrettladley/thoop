package xsync

import (
	"context"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xslog"
)

// refreshCurrent fetches the current cycle and the previous (n-1) cycle.
// This is called on TUI startup to ensure fresh data for display.
func (s *Service) refreshCurrent(ctx context.Context) error {
	s.logger.InfoContext(ctx, "refreshing current cycles")

	// Fetch the 2 most recent cycles
	resp, err := s.client.Cycle.List(ctx, &whoop.ListParams{Limit: 2})
	if err != nil {
		return err
	}

	if len(resp.Records) == 0 {
		s.logger.InfoContext(ctx, "no cycles found")
		return nil
	}

	// Save cycles
	if err := s.repo.Cycles.UpsertBatch(ctx, resp.Records); err != nil {
		return err
	}

	// Fetch recovery and sleep for each cycle in parallel
	for _, cycle := range resp.Records {
		// Fetch recovery
		if err := s.refreshRecoveryForCycle(ctx, cycle.ID); err != nil {
			s.logger.WarnContext(ctx, "failed to refresh recovery",
				xslog.CycleID(cycle.ID),
				xslog.Error(err))
		}

		// Fetch sleep
		if err := s.refreshSleepForCycle(ctx, cycle.ID); err != nil {
			s.logger.WarnContext(ctx, "failed to refresh sleep",
				xslog.CycleID(cycle.ID),
				xslog.Error(err))
		}
	}

	// Update last sync time
	now := time.Now()
	if err := s.repo.SyncState.UpdateLastFullSync(ctx, now); err != nil {
		s.logger.WarnContext(ctx, "failed to update last sync time", xslog.Error(err))
	}

	s.logger.InfoContext(ctx, "refreshed current cycles", xslog.Count(len(resp.Records)))
	return nil
}

// refreshRecoveryForCycle fetches and caches recovery for a cycle.
func (s *Service) refreshRecoveryForCycle(ctx context.Context, cycleID int64) error {
	recovery, err := s.client.Cycle.GetRecovery(ctx, cycleID)
	if err != nil {
		return err
	}
	return s.repo.Recoveries.Upsert(ctx, recovery)
}

// refreshSleepForCycle fetches and caches sleep for a cycle.
func (s *Service) refreshSleepForCycle(ctx context.Context, cycleID int64) error {
	sleep, err := s.client.Cycle.GetSleep(ctx, cycleID)
	if err != nil {
		return err
	}
	return s.repo.Sleeps.Upsert(ctx, sleep)
}
