package xsync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/repository"
)

type SyncService interface {
	// StartBackfill begins a background goroutine to fetch 1 month of historical data.
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

type Service struct {
	client *whoop.Client
	repo   *repository.Repository
	logger *slog.Logger
}

var _ SyncService = (*Service)(nil)

func NewService(client *whoop.Client, repo *repository.Repository, logger *slog.Logger) *Service {
	return &Service{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) StartBackfill(ctx context.Context) error {
	state, err := s.repo.SyncState.Get(ctx)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if state.BackfillComplete {
		s.logger.InfoContext(ctx, "backfill already complete, skipping")
		return nil
	}

	go s.runBackfill(ctx)
	return nil
}

func (s *Service) RefreshCurrent(ctx context.Context) error {
	return s.refreshCurrent(ctx)
}

func (s *Service) FetchHistorical(ctx context.Context, start, end time.Time) error {
	return s.fetchHistorical(ctx, start, end)
}

func (s *Service) IsBackfillComplete(ctx context.Context) (bool, error) {
	state, err := s.repo.SyncState.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("%w", err)
	}
	return state.BackfillComplete, nil
}
