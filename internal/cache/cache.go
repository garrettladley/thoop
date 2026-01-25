package cache

import (
	"context"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

// CacheService abstracts all caching decisions from the TUI.
// It wraps repository + API client, handling staleness, coverage detection,
// and deciding when to use cached data vs. fetching from the API.
type CacheService interface {
	// Single-date operations
	GetCycleForDate(ctx context.Context, date time.Time) (*CacheResult[*whoop.Cycle], error)
	GetRecoveryForCycle(ctx context.Context, cycleID int64) (*CacheResult[*whoop.Recovery], error)
	GetSleepForCycle(ctx context.Context, cycleID int64) (*CacheResult[*whoop.Sleep], error)
	GetWorkoutsForDateRange(ctx context.Context, start, end time.Time) (*CacheResult[[]whoop.Workout], error)

	// Range operations (for charts, historical data)
	GetCyclesForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Cycle], error)
	GetRecoveriesForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Recovery], error)
	GetSleepsForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Sleep], error)

	// Calendar-specific (fetches recoveries and cycles for proper date mapping)
	GetCalendarData(ctx context.Context, month time.Time) (*CalendarData, error)

	// Historical bundle (30-day data for charts)
	GetHistoricalData(ctx context.Context, referenceDate time.Time, days int) (*HistoricalData, error)

	// SetAPIKey updates the API key on the underlying client (called after OAuth flow)
	SetAPIKey(apiKey string)
}

// CacheResult wraps a single cached item with metadata about its source.
type CacheResult[T any] struct {
	Data      T
	FromCache bool
}

// DateRangeResult wraps a collection of records with cache metadata.
type DateRangeResult[T any] struct {
	Records      []T
	FromCache    bool
	PartialCache bool // some from cache, some from API
}

// HistoricalData bundles multiple data types for chart display.
type HistoricalData struct {
	Cycles     []whoop.Cycle
	Recoveries []whoop.Recovery
	Sleeps     []whoop.Sleep
	Workouts   []whoop.Workout
	FromCache  bool
}

// CalendarData bundles recoveries and cycles for calendar display.
type CalendarData struct {
	Recoveries []whoop.Recovery
	Cycles     []whoop.Cycle
	FromCache  bool
}
