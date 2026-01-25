package cache

import (
	"testing"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

func TestDefaultStalenessChecker_ShouldRefresh(t *testing.T) {
	t.Parallel()

	checker := NewDefaultStalenessChecker()
	now := time.Now()

	tests := []struct {
		name       string
		scoreState whoop.ScoreState
		dataDate   time.Time
		fetchedAt  time.Time
		want       bool
	}{
		{
			name:       "unscorable never refreshes",
			scoreState: whoop.ScoreStateUnscorable,
			dataDate:   now,
			fetchedAt:  now.Add(-24 * time.Hour),
			want:       false,
		},
		{
			name:       "pending refreshes after 5 minutes",
			scoreState: whoop.ScoreStatePendingScore,
			dataDate:   now,
			fetchedAt:  now.Add(-6 * time.Minute),
			want:       true,
		},
		{
			name:       "pending does not refresh within 5 minutes",
			scoreState: whoop.ScoreStatePendingScore,
			dataDate:   now,
			fetchedAt:  now.Add(-3 * time.Minute),
			want:       false,
		},
		{
			name:       "scored today refreshes after 15 minutes",
			scoreState: whoop.ScoreStateScored,
			dataDate:   now,
			fetchedAt:  now.Add(-20 * time.Minute),
			want:       true,
		},
		{
			name:       "scored today does not refresh within 15 minutes",
			scoreState: whoop.ScoreStateScored,
			dataDate:   now,
			fetchedAt:  now.Add(-10 * time.Minute),
			want:       false,
		},
		{
			name:       "scored historical refreshes after 24 hours",
			scoreState: whoop.ScoreStateScored,
			dataDate:   now.AddDate(0, 0, -7),
			fetchedAt:  now.Add(-25 * time.Hour),
			want:       true,
		},
		{
			name:       "scored historical does not refresh within 24 hours",
			scoreState: whoop.ScoreStateScored,
			dataDate:   now.AddDate(0, 0, -7),
			fetchedAt:  now.Add(-12 * time.Hour),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checker.ShouldRefresh(tt.scoreState, tt.dataDate, tt.fetchedAt)
			if got != tt.want {
				t.Errorf("ShouldRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeverStaleChecker(t *testing.T) {
	t.Parallel()
	checker := &NeverStaleChecker{}
	now := time.Now()

	// should always return false regardless of inputs
	if checker.ShouldRefresh(whoop.ScoreStatePendingScore, now, now.Add(-24*time.Hour)) {
		t.Error("NeverStaleChecker should always return false")
	}
}

func TestAlwaysStaleChecker(t *testing.T) {
	t.Parallel()
	checker := &AlwaysStaleChecker{}
	now := time.Now()

	// should always return true regardless of inputs
	if !checker.ShouldRefresh(whoop.ScoreStateScored, now, now) {
		t.Error("AlwaysStaleChecker should always return true")
	}
}

func TestService_hasSufficientCoverage(t *testing.T) {
	t.Parallel()
	svc := &Service{}

	tests := []struct {
		name        string
		start       time.Time
		end         time.Time
		cachedDates map[string]bool
		want        bool
	}{
		{
			name:        "nil cached dates",
			start:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			cachedDates: nil,
			want:        false,
		},
		{
			name:        "empty cached dates",
			start:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			cachedDates: map[string]bool{},
			want:        false,
		},
		{
			name:  "80% coverage met",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC), // 10 days
			cachedDates: map[string]bool{
				"2024-01-01": true,
				"2024-01-02": true,
				"2024-01-03": true,
				"2024-01-04": true,
				"2024-01-05": true,
				"2024-01-06": true,
				"2024-01-07": true,
				"2024-01-08": true, // 8 of 10 = 80%
			},
			want: true,
		},
		{
			name:  "below 80% coverage",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC), // 10 days
			cachedDates: map[string]bool{
				"2024-01-01": true,
				"2024-01-02": true,
				"2024-01-03": true,
				"2024-01-04": true,
				"2024-01-05": true,
				"2024-01-06": true,
				"2024-01-07": true, // 7 of 10 = 70%
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := svc.hasSufficientCoverage(tt.start, tt.end, tt.cachedDates)
			if got != tt.want {
				t.Errorf("hasSufficientCoverage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_findMissingRanges(t *testing.T) {
	t.Parallel()
	svc := &Service{}

	tests := []struct {
		name        string
		start       time.Time
		end         time.Time
		cachedDates map[string]bool
		wantLen     int
	}{
		{
			name:        "no cache returns full range",
			start:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			cachedDates: nil,
			wantLen:     1,
		},
		{
			name:  "gap in middle",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC),
			cachedDates: map[string]bool{
				"2024-01-01": true,
				"2024-01-02": true,
				// gap: 2024-01-03, 2024-01-04
				"2024-01-05": true,
			},
			wantLen: 1, // one gap range
		},
		{
			name:  "multiple gaps",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			cachedDates: map[string]bool{
				"2024-01-02": true,
				"2024-01-04": true,
				"2024-01-06": true,
			},
			wantLen: 4, // gaps: 1, 3, 5, 7
		},
		{
			name:  "all cached",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
			cachedDates: map[string]bool{
				"2024-01-01": true,
				"2024-01-02": true,
				"2024-01-03": true,
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := svc.findMissingRanges(tt.start, tt.end, tt.cachedDates)
			if len(got) != tt.wantLen {
				t.Errorf("findMissingRanges() returned %d ranges, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestMergeRecoverySlices(t *testing.T) {
	t.Parallel()
	r1 := whoop.Recovery{CycleID: 1}
	r2 := whoop.Recovery{CycleID: 2}
	r3 := whoop.Recovery{CycleID: 3}
	r1Dup := whoop.Recovery{CycleID: 1} // duplicate

	tests := []struct {
		name    string
		a       []whoop.Recovery
		b       []whoop.Recovery
		wantLen int
	}{
		{
			name:    "no overlap",
			a:       []whoop.Recovery{r1, r2},
			b:       []whoop.Recovery{r3},
			wantLen: 3,
		},
		{
			name:    "with duplicates",
			a:       []whoop.Recovery{r1, r2},
			b:       []whoop.Recovery{r1Dup, r3},
			wantLen: 3, // r1 only counted once
		},
		{
			name:    "empty a",
			a:       []whoop.Recovery{},
			b:       []whoop.Recovery{r1, r2},
			wantLen: 2,
		},
		{
			name:    "empty b",
			a:       []whoop.Recovery{r1, r2},
			b:       []whoop.Recovery{},
			wantLen: 2,
		},
		{
			name:    "both empty",
			a:       []whoop.Recovery{},
			b:       []whoop.Recovery{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MergeRecoverySlices(tt.a, tt.b)
			if len(got) != tt.wantLen {
				t.Errorf("MergeRecoverySlices() returned %d items, want %d", len(got), tt.wantLen)
			}
		})
	}
}
