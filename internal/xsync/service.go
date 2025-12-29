package xsync

import (
	"context"
	"log/slog"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/repository"
)

// SyncService coordinates data synchronization between WHOOP API and local cache.
type SyncService interface {
	// StartBackfill begins a background goroutine to fetch 1 month of historical data.
	// Non-blocking; returns immediately.
	StartBackfill(ctx context.Context) error

	// RefreshCurrent fetches the current cycle and n-1 cycle from the API.
	// This should be called on TUI startup.
	RefreshCurrent(ctx context.Context) error

	// FetchHistorical fetches data for a specific date range from the API.
	// Used for on-demand fetching when user expands the time horizon.
	FetchHistorical(ctx context.Context, start, end time.Time) error

	// IsBackfillComplete returns whether the initial backfill has finished.
	IsBackfillComplete(ctx context.Context) (bool, error)
}

// Service implements SyncService.
type Service struct {
	client *whoop.Client
	repo   *repository.Repository
	logger *slog.Logger
}

// NewService creates a new sync service.
func NewService(client *whoop.Client, repo *repository.Repository, logger *slog.Logger) *Service {
	return &Service{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

// StartBackfill starts background data backfill.
func (s *Service) StartBackfill(ctx context.Context) error {
	state, err := s.repo.SyncState.Get(ctx)
	if err != nil {
		return err
	}

	if state.BackfillComplete {
		s.logger.InfoContext(ctx, "backfill already complete, skipping")
		return nil
	}

	go s.runBackfill(ctx)
	return nil
}

// RefreshCurrent fetches current + n-1 cycle.
func (s *Service) RefreshCurrent(ctx context.Context) error {
	return s.refreshCurrent(ctx)
}

// FetchHistorical fetches data for a date range.
func (s *Service) FetchHistorical(ctx context.Context, start, end time.Time) error {
	return s.fetchHistorical(ctx, start, end)
}

// IsBackfillComplete checks if backfill is done.
func (s *Service) IsBackfillComplete(ctx context.Context) (bool, error) {
	state, err := s.repo.SyncState.Get(ctx)
	if err != nil {
		return false, err
	}
	return state.BackfillComplete, nil
}
