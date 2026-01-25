package cache

import (
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xtime"
)

// StalenessChecker determines whether cached data should be refreshed.
type StalenessChecker interface {
	// ShouldRefresh returns true if the cached data is stale and should be re-fetched.
	ShouldRefresh(scoreState whoop.ScoreState, dataDate, fetchedAt time.Time) bool
}

// DefaultStalenessChecker implements time-based staleness rules:
// - UNSCORABLE: never refresh (data is final)
// - PENDING_SCORE: refresh after 5 minutes (waiting for scoring)
// - SCORED + today: refresh after 15 minutes (might have updates)
// - SCORED + historical: refresh after 24 hours (very stable)
type DefaultStalenessChecker struct {
	pendingRefreshInterval  time.Duration
	todayScoredRefreshAfter time.Duration
	historicalRefreshAfter  time.Duration
}

// NewDefaultStalenessChecker creates a StalenessChecker with sensible defaults.
func NewDefaultStalenessChecker() *DefaultStalenessChecker {
	return &DefaultStalenessChecker{
		pendingRefreshInterval:  5 * time.Minute,
		todayScoredRefreshAfter: 15 * time.Minute,
		historicalRefreshAfter:  24 * time.Hour,
	}
}

// ShouldRefresh implements StalenessChecker.
func (c *DefaultStalenessChecker) ShouldRefresh(scoreState whoop.ScoreState, dataDate, fetchedAt time.Time) bool {
	now := time.Now()
	age := now.Sub(fetchedAt)

	switch scoreState {
	case whoop.ScoreStateUnscorable:
		// unscorable data never changes
		return false

	case whoop.ScoreStatePendingScore:
		// pending data might be scored soon, check frequently
		return age > c.pendingRefreshInterval

	case whoop.ScoreStateScored:
		// scored data is stable, but today's data might still update
		if xtime.SameDay(dataDate, now) {
			return age > c.todayScoredRefreshAfter
		}
		// historical scored data is very stable
		return age > c.historicalRefreshAfter

	default:
		// unknown state, assume stale
		return true
	}
}

type NeverStaleChecker struct{}

func (c *NeverStaleChecker) ShouldRefresh(scoreState whoop.ScoreState, dataDate, fetchedAt time.Time) bool {
	return false
}

type AlwaysStaleChecker struct{}

func (c *AlwaysStaleChecker) ShouldRefresh(scoreState whoop.ScoreState, dataDate, fetchedAt time.Time) bool {
	return true
}
