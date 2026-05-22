package cache

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/xtime"
)

// coverageThreshold is the percentage of expected days that must be cached
// to consider the cache "sufficient" (80% coverage).
const coverageThreshold = 0.80

// Service implements CacheService by wrapping a repository and API client.
type Service struct {
	client           *whoop.Client
	repo             *repository.Repository
	stalenessChecker StalenessChecker
}

// NewService creates a CacheService with the given dependencies.
func NewService(client *whoop.Client, repo *repository.Repository) *Service {
	return &Service{
		client:           client,
		repo:             repo,
		stalenessChecker: NewDefaultStalenessChecker(),
	}
}

// WithStalenessChecker sets a custom staleness checker.
func (s *Service) WithStalenessChecker(checker StalenessChecker) *Service {
	s.stalenessChecker = checker
	return s
}

// SetAPIKey updates the API key on the underlying whoop client.
func (s *Service) SetAPIKey(apiKey string) {
	if s.client != nil {
		s.client.SetAPIKey(apiKey)
	}
}

// GetCycleForDate fetches the cycle for a date, checking cache first.
func (s *Service) GetCycleForDate(ctx context.Context, date time.Time) (*CacheResult[*whoop.Cycle], error) {
	// 1. try cache first
	if s.repo != nil {
		cached, err := s.repo.Cycles.GetByDate(ctx, date)
		if err == nil && cached != nil {
			// check staleness based on score state and time
			if !s.stalenessChecker.ShouldRefresh(cached.ScoreState, cached.Start, cached.UpdatedAt) {
				return &CacheResult[*whoop.Cycle]{Data: cached, FromCache: true}, nil
			}
		}
	}

	// 2. cache miss or stale - fetch from API
	if s.client == nil {
		return &CacheResult[*whoop.Cycle]{Data: nil, FromCache: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := xtime.StartOfDay(date)
	end := start.Add(24 * time.Hour)

	cycles, err := s.client.Cycle.List(ctx, &whoop.ListParams{
		Limit: 1,
		Start: &start,
		End:   &end,
	})
	if err != nil {
		return nil, fmt.Errorf("list cycles: %w", err)
	}

	if len(cycles.Records) == 0 {
		return &CacheResult[*whoop.Cycle]{Data: nil, FromCache: false}, nil
	}

	cycle := &cycles.Records[0]

	// 3. store to cache for future use
	if s.repo != nil {
		_ = s.repo.Cycles.Upsert(ctx, cycle)
	}

	return &CacheResult[*whoop.Cycle]{Data: cycle, FromCache: false}, nil
}

// GetRecoveryForCycle fetches a recovery by cycle ID.
func (s *Service) GetRecoveryForCycle(ctx context.Context, cycleID int64) (*CacheResult[*whoop.Recovery], error) {
	// 1. try cache first
	if s.repo != nil {
		cached, err := s.repo.Recoveries.Get(ctx, cycleID)
		if err == nil && cached != nil {
			if !s.stalenessChecker.ShouldRefresh(cached.ScoreState, cached.CreatedAt, cached.UpdatedAt) {
				return &CacheResult[*whoop.Recovery]{Data: cached, FromCache: true}, nil
			}
		}
	}

	// 2. cache miss or stale - fetch from API
	if s.client == nil {
		return &CacheResult[*whoop.Recovery]{Data: nil, FromCache: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	recovery, err := s.client.Cycle.GetRecovery(ctx, cycleID)
	if err != nil {
		return nil, fmt.Errorf("get recovery for cycle %d: %w", cycleID, err)
	}

	// 3. store to cache
	if s.repo != nil && recovery != nil {
		_ = s.repo.Recoveries.Upsert(ctx, recovery)
	}

	return &CacheResult[*whoop.Recovery]{Data: recovery, FromCache: false}, nil
}

// GetSleepForCycle fetches sleep data for a cycle ID.
func (s *Service) GetSleepForCycle(ctx context.Context, cycleID int64) (*CacheResult[*whoop.Sleep], error) {
	// 1. try cache first
	if s.repo != nil {
		cached, err := s.repo.Sleeps.GetByCycleID(ctx, cycleID)
		if err == nil && cached != nil {
			if !s.stalenessChecker.ShouldRefresh(cached.ScoreState, cached.Start, cached.UpdatedAt) {
				return &CacheResult[*whoop.Sleep]{Data: cached, FromCache: true}, nil
			}
		}
	}

	// 2. Ccche miss or stale - fetch from API
	if s.client == nil {
		return &CacheResult[*whoop.Sleep]{Data: nil, FromCache: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sleep, err := s.client.Cycle.GetSleep(ctx, cycleID)
	if err != nil {
		return nil, fmt.Errorf("get sleep for cycle %d: %w", cycleID, err)
	}

	// 3. store to cache
	if s.repo != nil && sleep != nil {
		_ = s.repo.Sleeps.Upsert(ctx, sleep)
	}

	return &CacheResult[*whoop.Sleep]{Data: sleep, FromCache: false}, nil
}

// GetWorkoutsForDateRange fetches workouts within a date range.
func (s *Service) GetWorkoutsForDateRange(ctx context.Context, start, end time.Time) (*CacheResult[[]whoop.Workout], error) {
	// 1. try cache first
	if cached := s.tryGetWorkoutsFromCache(ctx, start, end); cached != nil {
		return cached, nil
	}

	// 2. fetch from API
	if s.client == nil {
		return &CacheResult[[]whoop.Workout]{Data: nil, FromCache: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var allWorkouts []whoop.Workout
	var nextToken *string

	for {
		resp, err := s.client.Workout.List(ctx, &whoop.ListParams{
			Limit:     25,
			Start:     &start,
			End:       &end,
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list workouts: %w", err)
		}
		allWorkouts = append(allWorkouts, resp.Records...)
		if !resp.HasMore() {
			break
		}
		nextToken = resp.NextToken
	}

	// 3. store to cache
	if s.repo != nil && len(allWorkouts) > 0 {
		_ = s.repo.Workouts.UpsertBatch(ctx, allWorkouts)
	}

	// 4. sort by start time (earliest first) to match SQL ordering
	slices.SortFunc(allWorkouts, func(a, b whoop.Workout) int {
		return a.Start.Compare(b.Start)
	})

	return &CacheResult[[]whoop.Workout]{Data: allWorkouts, FromCache: false}, nil
}

// GetCyclesForRange fetches cycles within a date range.
func (s *Service) GetCyclesForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Cycle], error) {
	return s.fetchRangeWithCache(ctx, start, end)
}

// GetRecoveriesForRange fetches recoveries within a date range.
func (s *Service) GetRecoveriesForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Recovery], error) {
	return s.fetchRecoveriesForRange(ctx, start, end)
}

// GetSleepsForRange fetches sleeps within a date range.
func (s *Service) GetSleepsForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Sleep], error) {
	return s.fetchSleepsForRange(ctx, start, end)
}

// GetCalendarRecoveries fetches recovery data for a calendar month.
// This is optimized to only fetch recoveries (not cycles), since
// the calendar only needs recovery scores for coloring days.
func (s *Service) GetCalendarData(ctx context.Context, month time.Time) (*CalendarData, error) {
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	// 1. check cache for existing data
	cachedRecoveries, cachedCycles, cachedDates := s.loadCalendarCache(ctx, startOfMonth, endOfMonth)

	// 2. find missing date ranges (always check - don't skip based on coverage percentage)
	missingRanges := s.findMissingRanges(startOfMonth, endOfMonth, cachedDates)

	// 3. fetch missing data from API
	if s.client == nil {
		if len(cachedRecoveries) > 0 {
			return &CalendarData{Recoveries: cachedRecoveries, Cycles: cachedCycles, FromCache: true}, nil
		}
		return &CalendarData{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var fetchedRecoveries []whoop.Recovery
	var fetchedCycles []whoop.Cycle

	for _, r := range missingRanges {
		var (
			rangeCycles     []whoop.Cycle
			rangeRecoveries []whoop.Recovery
			wg              sync.WaitGroup
		)

		wg.Add(2)

		go func(start, end time.Time) {
			defer wg.Done()
			var cycleToken *string
			for {
				params := &whoop.ListParams{
					Limit:     25,
					Start:     &start,
					End:       &end,
					NextToken: cycleToken,
				}
				resp, err := s.client.Cycle.List(ctx, params)
				if err != nil {
					return
				}
				rangeCycles = append(rangeCycles, resp.Records...)
				if !resp.HasMore() {
					return
				}
				cycleToken = resp.NextToken
			}
		}(r.start, r.end)

		go func(start, end time.Time) {
			defer wg.Done()
			var recoveryToken *string
			for {
				params := &whoop.ListParams{
					Limit:     25,
					Start:     &start,
					End:       &end,
					NextToken: recoveryToken,
				}
				resp, err := s.client.Recovery.List(ctx, params)
				if err != nil {
					return
				}
				rangeRecoveries = append(rangeRecoveries, resp.Records...)
				if !resp.HasMore() {
					return
				}
				recoveryToken = resp.NextToken
			}
		}(r.start, r.end)

		wg.Wait()

		fetchedCycles = append(fetchedCycles, rangeCycles...)
		fetchedRecoveries = append(fetchedRecoveries, rangeRecoveries...)
	}

	// 4. store fetched data to cache
	if s.repo != nil {
		if len(fetchedCycles) > 0 {
			_ = s.repo.Cycles.UpsertBatch(ctx, fetchedCycles)
		}
		if len(fetchedRecoveries) > 0 {
			_ = s.repo.Recoveries.UpsertBatch(ctx, fetchedRecoveries)
		}
	}

	// 5. merge cached and fetched data
	allRecoveries := MergeRecoverySlices(cachedRecoveries, fetchedRecoveries)
	allCycles := MergeCycleSlices(cachedCycles, fetchedCycles)

	// if we got nothing and had cached data, return cached
	if len(allRecoveries) == 0 && len(cachedRecoveries) > 0 {
		return &CalendarData{Recoveries: cachedRecoveries, Cycles: cachedCycles, FromCache: true}, nil
	}

	return &CalendarData{
		Recoveries: allRecoveries,
		Cycles:     allCycles,
		FromCache:  len(fetchedRecoveries) == 0,
	}, nil
}

// GetHistoricalData fetches bundled historical data for charts.
func (s *Service) GetHistoricalData(ctx context.Context, referenceDate time.Time, days int) (*HistoricalData, error) {
	startOfDay := xtime.StartOfDay(referenceDate)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startDate := startOfDay.AddDate(0, 0, -days)

	// 1. load all cached data from SQLite
	cachedCycles, cachedRecoveries, cachedSleeps := s.loadHistoricalCache(ctx, startDate, endOfDay)

	// 2. check if all cached data is fresh
	allFresh := len(cachedCycles) > 0
	for _, c := range cachedCycles {
		if s.stalenessChecker.ShouldRefresh(c.ScoreState, c.Start, c.UpdatedAt) {
			allFresh = false
			break
		}
	}
	for _, r := range cachedRecoveries {
		if s.stalenessChecker.ShouldRefresh(r.ScoreState, r.CreatedAt, r.UpdatedAt) {
			allFresh = false
			break
		}
	}
	for _, sl := range cachedSleeps {
		if s.stalenessChecker.ShouldRefresh(sl.ScoreState, sl.Start, sl.UpdatedAt) {
			allFresh = false
			break
		}
	}

	// if we have sufficient fresh data, use cache
	minDaysRequired := max(7, days/4) // at least 7 days or 25% of requested days
	if allFresh && len(cachedCycles) >= minDaysRequired {
		return &HistoricalData{
			Cycles:     cachedCycles,
			Recoveries: cachedRecoveries,
			Sleeps:     cachedSleeps,
			FromCache:  true,
		}, nil
	}

	// 3. cache incomplete - fetch from API
	if s.client == nil {
		return &HistoricalData{
			Cycles:     cachedCycles,
			Recoveries: cachedRecoveries,
			Sleeps:     cachedSleeps,
			FromCache:  true,
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var (
		allRecoveries []whoop.Recovery
		allCycles     []whoop.Cycle
		allSleeps     []whoop.Sleep
		recoveryErr   error
		cycleErr      error
		sleepErr      error
		wg            sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		var nextToken *string
		for {
			params := &whoop.ListParams{
				Limit:     25,
				Start:     &startDate,
				End:       &endOfDay,
				NextToken: nextToken,
			}
			resp, err := s.client.Cycle.List(ctx, params)
			if err != nil {
				cycleErr = err
				return
			}
			allCycles = append(allCycles, resp.Records...)
			if !resp.HasMore() {
				return
			}
			nextToken = resp.NextToken
		}
	}()

	go func() {
		defer wg.Done()
		var nextToken *string
		for {
			params := &whoop.ListParams{
				Limit:     25,
				Start:     &startDate,
				End:       &endOfDay,
				NextToken: nextToken,
			}
			resp, err := s.client.Sleep.List(ctx, params)
			if err != nil {
				sleepErr = err
				return
			}
			allSleeps = append(allSleeps, resp.Records...)
			if !resp.HasMore() {
				return
			}
			nextToken = resp.NextToken
		}
	}()

	// fetch recoveries in main goroutine
	var nextToken *string
	for {
		params := &whoop.ListParams{
			Limit:     25,
			Start:     &startDate,
			End:       &endOfDay,
			NextToken: nextToken,
		}
		resp, err := s.client.Recovery.List(ctx, params)
		if err != nil {
			recoveryErr = err
			break
		}
		allRecoveries = append(allRecoveries, resp.Records...)
		if !resp.HasMore() {
			break
		}
		nextToken = resp.NextToken
	}

	wg.Wait()

	// return first error encountered
	if recoveryErr != nil {
		return nil, recoveryErr
	}
	if cycleErr != nil {
		return nil, cycleErr
	}
	if sleepErr != nil {
		return nil, sleepErr
	}

	// 4. store fetched data to SQLite
	if s.repo != nil {
		_ = s.repo.Cycles.UpsertBatch(ctx, allCycles)
		_ = s.repo.Sleeps.UpsertBatch(ctx, allSleeps)
		_ = s.repo.Recoveries.UpsertBatch(ctx, allRecoveries)
	}

	return &HistoricalData{
		Cycles:     allCycles,
		Recoveries: allRecoveries,
		Sleeps:     allSleeps,
		FromCache:  false,
	}, nil
}

// tryGetWorkoutsFromCache attempts to load fresh workouts from cache.
func (s *Service) tryGetWorkoutsFromCache(ctx context.Context, start, end time.Time) *CacheResult[[]whoop.Workout] {
	if s.repo == nil {
		return nil
	}

	workouts, err := collectCursorRecords(ctx, func(cursor *time.Time) (*repository.CursorResult[whoop.Workout], error) {
		return s.repo.Workouts.GetByDateRange(ctx, start, end, &repository.CursorParams{Limit: repository.DefaultPageSize, Cursor: cursor})
	})
	if err != nil || len(workouts) == 0 {
		return nil
	}

	for _, w := range workouts {
		if s.stalenessChecker.ShouldRefresh(w.ScoreState, w.Start, w.UpdatedAt) {
			return nil
		}
	}

	return &CacheResult[[]whoop.Workout]{Data: workouts, FromCache: true}
}

// loadCalendarCache loads cached cycles and recoveries for calendar display.
func (s *Service) loadCalendarCache(ctx context.Context, start, end time.Time) ([]whoop.Recovery, []whoop.Cycle, map[string]bool) {
	if s.repo == nil {
		return nil, nil, nil
	}

	cycles, err := collectCursorRecords(ctx, func(cursor *time.Time) (*repository.CursorResult[whoop.Cycle], error) {
		return s.repo.Cycles.GetByDateRange(ctx, start, end, &repository.CursorParams{Limit: repository.DefaultPageSize, Cursor: cursor})
	})
	if err != nil || len(cycles) == 0 {
		return nil, nil, nil
	}

	cycleIDs := make([]int64, len(cycles))
	cycleStartByID := make(map[int64]time.Time, len(cycles))

	for i, c := range cycles {
		cycleIDs[i] = c.ID
		cycleStartByID[c.ID] = c.Start
	}

	recoveries, err := s.repo.Recoveries.GetByCycleIDs(ctx, cycleIDs)
	if err != nil {
		return nil, cycles, nil
	}

	// only mark dates as cached if we have BOTH cycle AND recovery
	cachedDates := make(map[string]bool)
	for _, r := range recoveries {
		if cycleStart, ok := cycleStartByID[r.CycleID]; ok {
			dateKey := cycleStart.Format("2006-01-02")
			cachedDates[dateKey] = true
		}
	}

	return recoveries, cycles, cachedDates
}

// loadHistoricalCache loads all cached historical data.
func (s *Service) loadHistoricalCache(ctx context.Context, start, end time.Time) ([]whoop.Cycle, []whoop.Recovery, []whoop.Sleep) {
	if s.repo == nil {
		return nil, nil, nil
	}

	var cachedCycles []whoop.Cycle
	var cachedRecoveries []whoop.Recovery
	var cachedSleeps []whoop.Sleep

	cycles, err := collectCursorRecords(ctx, func(cursor *time.Time) (*repository.CursorResult[whoop.Cycle], error) {
		return s.repo.Cycles.GetByDateRange(ctx, start, end, &repository.CursorParams{Limit: repository.DefaultPageSize, Cursor: cursor})
	})
	if err == nil {
		cachedCycles = cycles
	}

	sleeps, err := collectCursorRecords(ctx, func(cursor *time.Time) (*repository.CursorResult[whoop.Sleep], error) {
		return s.repo.Sleeps.GetByDateRange(ctx, start, end, &repository.CursorParams{Limit: repository.DefaultPageSize, Cursor: cursor})
	})
	if err == nil {
		cachedSleeps = sleeps
	}

	if len(cachedCycles) > 0 {
		cycleIDs := make([]int64, len(cachedCycles))
		for i, c := range cachedCycles {
			cycleIDs[i] = c.ID
		}
		if recoveries, err := s.repo.Recoveries.GetByCycleIDs(ctx, cycleIDs); err == nil {
			cachedRecoveries = recoveries
		}
	}

	return cachedCycles, cachedRecoveries, cachedSleeps
}

// tryGetCyclesFromCache attempts to load fresh cycles from cache.
func (s *Service) tryGetCyclesFromCache(ctx context.Context, start, end time.Time) *DateRangeResult[whoop.Cycle] {
	if s.repo == nil {
		return nil
	}

	cycles, err := collectCursorRecords(ctx, func(cursor *time.Time) (*repository.CursorResult[whoop.Cycle], error) {
		return s.repo.Cycles.GetByDateRange(ctx, start, end, &repository.CursorParams{Limit: repository.DefaultPageSize, Cursor: cursor})
	})
	if err != nil || len(cycles) == 0 {
		return nil
	}

	for _, c := range cycles {
		if s.stalenessChecker.ShouldRefresh(c.ScoreState, c.Start, c.UpdatedAt) {
			return nil
		}
	}

	return &DateRangeResult[whoop.Cycle]{Records: cycles, FromCache: true}
}

// tryGetRecoveriesFromCache attempts to load fresh recoveries from cache.
func (s *Service) tryGetRecoveriesFromCache(ctx context.Context, cycleIDs []int64) *DateRangeResult[whoop.Recovery] {
	if s.repo == nil {
		return nil
	}

	recoveries, err := s.repo.Recoveries.GetByCycleIDs(ctx, cycleIDs)
	if err != nil || len(recoveries) == 0 {
		return nil
	}

	for _, r := range recoveries {
		if s.stalenessChecker.ShouldRefresh(r.ScoreState, r.CreatedAt, r.UpdatedAt) {
			return nil
		}
	}

	// require 80% coverage
	if len(recoveries) < len(cycleIDs)*8/10 {
		return nil
	}

	return &DateRangeResult[whoop.Recovery]{Records: recoveries, FromCache: true}
}

// tryGetSleepsFromCache attempts to load fresh sleeps from cache.
func (s *Service) tryGetSleepsFromCache(ctx context.Context, start, end time.Time) *DateRangeResult[whoop.Sleep] {
	if s.repo == nil {
		return nil
	}

	sleeps, err := collectCursorRecords(ctx, func(cursor *time.Time) (*repository.CursorResult[whoop.Sleep], error) {
		return s.repo.Sleeps.GetByDateRange(ctx, start, end, &repository.CursorParams{Limit: repository.DefaultPageSize, Cursor: cursor})
	})
	if err != nil || len(sleeps) == 0 {
		return nil
	}

	for _, sl := range sleeps {
		if s.stalenessChecker.ShouldRefresh(sl.ScoreState, sl.Start, sl.UpdatedAt) {
			return nil
		}
	}

	return &DateRangeResult[whoop.Sleep]{Records: sleeps, FromCache: true}
}

func collectCursorRecords[T any](ctx context.Context, fetch func(*time.Time) (*repository.CursorResult[T], error)) ([]T, error) {
	var records []T
	var cursor *time.Time

	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("collect cursor records: %w", err)
		}

		result, err := fetch(cursor)
		if err != nil {
			return records, fmt.Errorf("fetch cursor page: %w", err)
		}
		if result == nil {
			return records, nil
		}
		records = append(records, result.Records...)
		if result.NextCursor == nil {
			return records, nil
		}
		cursor = result.NextCursor
	}
}

// hasSufficientCoverage checks if cached data covers enough of the date range.
// Uses percentage-based coverage (>80%) instead of arbitrary magic numbers.
func (s *Service) hasSufficientCoverage(start, end time.Time, cachedDates map[string]bool) bool {
	if len(cachedDates) == 0 {
		return false
	}

	// calculate expected days in range (capped at today if range extends into future)
	now := time.Now()
	if end.After(now) {
		end = now
	}

	expectedDays := 0
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		expectedDays++
	}

	if expectedDays == 0 {
		return true // empty range is "covered"
	}

	coverage := float64(len(cachedDates)) / float64(expectedDays)
	return coverage >= coverageThreshold
}

type dateRange struct {
	start, end time.Time
}

// findMissingRanges identifies date ranges not covered by cached data.
func (s *Service) findMissingRanges(start, end time.Time, cachedDates map[string]bool) []dateRange {
	if len(cachedDates) == 0 {
		return []dateRange{{start: start, end: end}}
	}

	var ranges []dateRange
	var rangeStart *time.Time

	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		isCached := cachedDates[dateKey]

		if !isCached && rangeStart == nil {
			t := d
			rangeStart = &t
		} else if isCached && rangeStart != nil {
			ranges = append(ranges, dateRange{start: *rangeStart, end: d})
			rangeStart = nil
		}
	}

	if rangeStart != nil {
		ranges = append(ranges, dateRange{start: *rangeStart, end: end})
	}

	return ranges
}

// fetchRangeWithCache fetches cycles for a range with caching.
func (s *Service) fetchRangeWithCache(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Cycle], error) {
	// 1. try cache
	if cached := s.tryGetCyclesFromCache(ctx, start, end); cached != nil {
		return cached, nil
	}

	// 2. fetch from API
	if s.client == nil {
		return &DateRangeResult[whoop.Cycle]{Records: nil, FromCache: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var allCycles []whoop.Cycle
	var nextToken *string

	for {
		params := &whoop.ListParams{
			Limit:     25,
			Start:     &start,
			End:       &end,
			NextToken: nextToken,
		}
		resp, err := s.client.Cycle.List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("list cycles: %w", err)
		}
		allCycles = append(allCycles, resp.Records...)
		if !resp.HasMore() {
			break
		}
		nextToken = resp.NextToken
	}

	// 3. store to cache
	if s.repo != nil && len(allCycles) > 0 {
		_ = s.repo.Cycles.UpsertBatch(ctx, allCycles)
	}

	return &DateRangeResult[whoop.Cycle]{Records: allCycles, FromCache: false}, nil
}

// fetchRecoveriesForRange fetches recoveries for a range with caching.
func (s *Service) fetchRecoveriesForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Recovery], error) {
	// for recoveries, we first need cycles to get cycle IDs
	cyclesResult, err := s.fetchRangeWithCache(ctx, start, end)
	if err != nil {
		return nil, err
	}

	if len(cyclesResult.Records) == 0 {
		return &DateRangeResult[whoop.Recovery]{Records: nil, FromCache: cyclesResult.FromCache}, nil
	}

	cycleIDs := make([]int64, len(cyclesResult.Records))
	for i, c := range cyclesResult.Records {
		cycleIDs[i] = c.ID
	}

	// try cache for recoveries
	if cached := s.tryGetRecoveriesFromCache(ctx, cycleIDs); cached != nil {
		return cached, nil
	}

	// fetch from API
	if s.client == nil {
		return &DateRangeResult[whoop.Recovery]{Records: nil, FromCache: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var allRecoveries []whoop.Recovery
	var nextToken *string

	for {
		params := &whoop.ListParams{
			Limit:     25,
			Start:     &start,
			End:       &end,
			NextToken: nextToken,
		}
		resp, err := s.client.Recovery.List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("list recoveries: %w", err)
		}
		allRecoveries = append(allRecoveries, resp.Records...)
		if !resp.HasMore() {
			break
		}
		nextToken = resp.NextToken
	}

	if s.repo != nil && len(allRecoveries) > 0 {
		_ = s.repo.Recoveries.UpsertBatch(ctx, allRecoveries)
	}

	return &DateRangeResult[whoop.Recovery]{Records: allRecoveries, FromCache: false}, nil
}

// fetchSleepsForRange fetches sleeps for a range with caching.
func (s *Service) fetchSleepsForRange(ctx context.Context, start, end time.Time) (*DateRangeResult[whoop.Sleep], error) {
	// 1. try cache
	if cached := s.tryGetSleepsFromCache(ctx, start, end); cached != nil {
		return cached, nil
	}

	// 2. fetch from API
	if s.client == nil {
		return &DateRangeResult[whoop.Sleep]{Records: nil, FromCache: false}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var allSleeps []whoop.Sleep
	var nextToken *string

	for {
		params := &whoop.ListParams{
			Limit:     25,
			Start:     &start,
			End:       &end,
			NextToken: nextToken,
		}
		resp, err := s.client.Sleep.List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("list sleeps: %w", err)
		}
		allSleeps = append(allSleeps, resp.Records...)
		if !resp.HasMore() {
			break
		}
		nextToken = resp.NextToken
	}

	// 3. store to cache
	if s.repo != nil && len(allSleeps) > 0 {
		_ = s.repo.Sleeps.UpsertBatch(ctx, allSleeps)
	}

	return &DateRangeResult[whoop.Sleep]{Records: allSleeps, FromCache: false}, nil
}

// MergeRecoverySlices combines two recovery slices, deduping by CycleID.
// MergeRecoverySlices merges two recovery slices, deduplicating by CycleID.
func MergeRecoverySlices(a, b []whoop.Recovery) []whoop.Recovery {
	seen := make(map[int64]struct{})
	result := make([]whoop.Recovery, 0, len(a)+len(b))

	for _, r := range a {
		if _, ok := seen[r.CycleID]; !ok {
			seen[r.CycleID] = struct{}{}
			result = append(result, r)
		}
	}
	for _, r := range b {
		if _, ok := seen[r.CycleID]; !ok {
			seen[r.CycleID] = struct{}{}
			result = append(result, r)
		}
	}

	return result
}

// MergeCycleSlices merges two cycle slices, deduplicating by ID.
func MergeCycleSlices(a, b []whoop.Cycle) []whoop.Cycle {
	seen := make(map[int64]struct{})
	result := make([]whoop.Cycle, 0, len(a)+len(b))

	for _, c := range a {
		if _, ok := seen[c.ID]; !ok {
			seen[c.ID] = struct{}{}
			result = append(result, c)
		}
	}
	for _, c := range b {
		if _, ok := seen[c.ID]; !ok {
			seen[c.ID] = struct{}{}
			result = append(result, c)
		}
	}

	return result
}
