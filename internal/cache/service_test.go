package cache

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/repository"
	"golang.org/x/oauth2"
)

const (
	testDate20240101 = "2024-01-01"
	testDate20240102 = "2024-01-02"
	testDate20240103 = "2024-01-03"
	testDate20240104 = "2024-01-04"
	testDate20240105 = "2024-01-05"
	testDate20240106 = "2024-01-06"
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

func TestCollectCursorRecordsDrainsAllPages(t *testing.T) {
	t.Parallel()

	pageStarts := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	calls := 0

	got, err := collectCursorRecords(t.Context(), func(cursor *time.Time) (*repository.CursorResult[string], error) {
		calls++
		if cursor == nil {
			return &repository.CursorResult[string]{
				Records:    []string{"a", "b"},
				NextCursor: &pageStarts[0],
			}, nil
		}
		if cursor.Equal(pageStarts[0]) {
			return &repository.CursorResult[string]{
				Records:    []string{"c"},
				NextCursor: &pageStarts[1],
			}, nil
		}
		return &repository.CursorResult[string]{
			Records: []string{"d"},
		}, nil
	})
	if err != nil {
		t.Fatalf("collectCursorRecords() error = %v", err)
	}
	if !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("collectCursorRecords() = %v", got)
	}
	if calls != 3 {
		t.Fatalf("fetch calls = %d, want 3", calls)
	}
}

func TestService_GetWorkoutsForDateRangeFollowsAPIPagination(t *testing.T) {
	t.Parallel()

	server, requests := newPaginatedWorkoutServer(t)

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client := whoop.New(tokenSource, whoop.WithProxyURL(server.URL), whoop.WithTimeout(time.Second))
	svc := NewService(client, nil)

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	result, err := svc.GetWorkoutsForDateRange(t.Context(), start, end)
	if err != nil {
		t.Fatalf("GetWorkoutsForDateRange() error = %v", err)
	}
	if result.FromCache {
		t.Fatal("FromCache = true, want false")
	}
	if got := len(result.Data); got != 2 {
		t.Fatalf("len(Data) = %d, want 2", got)
	}
	if gotIDs := []string{result.Data[0].ID, result.Data[1].ID}; !reflect.DeepEqual(gotIDs, []string{"workout-2", "workout-1"}) {
		t.Fatalf("workout IDs = %v", gotIDs)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}

func newPaginatedWorkoutServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/activity/workout" {
			t.Errorf("path = %s, want /v2/activity/workout", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("nextToken") {
		case "":
			if got := r.URL.Query().Get("limit"); got != "25" {
				t.Errorf("limit = %s, want 25", got)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"records": [
					{
						"id": "workout-1",
						"user_id": 1,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z",
						"start": "2024-01-01T02:00:00Z",
						"end": "2024-01-01T02:10:00Z",
						"timezone_offset": "+00:00",
						"sport_name": "walking",
						"score_state": "SCORED"
					}
				],
				"next_token": "page-2"
			}`))
		case "page-2":
			_, _ = w.Write([]byte(`{
				"records": [
					{
						"id": "workout-2",
						"user_id": 1,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z",
						"start": "2024-01-01T01:00:00Z",
						"end": "2024-01-01T01:10:00Z",
						"timezone_offset": "+00:00",
						"sport_name": "cycling",
						"score_state": "SCORED"
					}
				]
			}`))
		default:
			t.Errorf("unexpected nextToken %q", r.URL.Query().Get("nextToken"))
			http.Error(w, "bad next token", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	return server, &requests
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
				testDate20240101: true,
				testDate20240102: true,
				testDate20240103: true,
				testDate20240104: true,
				testDate20240105: true,
				testDate20240106: true,
				"2024-01-07":     true,
				"2024-01-08":     true, // 8 of 10 = 80%
			},
			want: true,
		},
		{
			name:  "below 80% coverage",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC), // 10 days
			cachedDates: map[string]bool{
				testDate20240101: true,
				testDate20240102: true,
				testDate20240103: true,
				testDate20240104: true,
				testDate20240105: true,
				testDate20240106: true,
				"2024-01-07":     true, // 7 of 10 = 70%
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
				testDate20240101: true,
				testDate20240102: true,
				// gap: 2024-01-03, 2024-01-04
				testDate20240105: true,
			},
			wantLen: 1, // one gap range
		},
		{
			name:  "multiple gaps",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			cachedDates: map[string]bool{
				testDate20240102: true,
				testDate20240104: true,
				testDate20240106: true,
			},
			wantLen: 4, // gaps: 1, 3, 5, 7
		},
		{
			name:  "all cached",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
			cachedDates: map[string]bool{
				testDate20240101: true,
				testDate20240102: true,
				testDate20240103: true,
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
