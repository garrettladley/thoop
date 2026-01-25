package dashboard

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/cache"
	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xtime"
)

// DateChangedMsg signals that the selected date has changed.
type DateChangedMsg struct {
	Date *time.Time
}

// CalendarSpinnerTickMsg triggers the next spinner animation frame.
type CalendarSpinnerTickMsg struct{}

// CalendarSpinnerTickCmd returns a command that ticks the spinner at 10 FPS.
func CalendarSpinnerTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return CalendarSpinnerTickMsg{}
	})
}

type CycleMsg struct {
	Cycle     *whoop.Cycle
	FromCache bool
	Err       error
}

// FetchCycleCmd fetches the most recent cycle (today's cycle).
func FetchCycleCmd(ctx context.Context, cacheSvc cache.CacheService) tea.Cmd {
	return FetchCycleForDateCmd(ctx, cacheSvc, time.Now())
}

// FetchCycleForDateCmd fetches the cycle for a given date via CacheService.
func FetchCycleForDateCmd(ctx context.Context, cacheSvc cache.CacheService, referenceDate time.Time) tea.Cmd {
	if cacheSvc == nil {
		return func() tea.Msg {
			return CycleMsg{Cycle: nil}
		}
	}

	return func() tea.Msg {
		result, err := cacheSvc.GetCycleForDate(ctx, referenceDate)
		if err != nil {
			return CycleMsg{Err: err}
		}
		if result == nil {
			return CycleMsg{Cycle: nil}
		}
		return CycleMsg{Cycle: result.Data, FromCache: result.FromCache}
	}
}

type SleepMsg struct {
	Sleep     *whoop.Sleep
	FromCache bool
	Err       error
}

// FetchSleepCmd fetches sleep data for a cycle via CacheService.
func FetchSleepCmd(ctx context.Context, cacheSvc cache.CacheService, cycleID int64) tea.Cmd {
	if cacheSvc == nil {
		return func() tea.Msg {
			return SleepMsg{Sleep: nil}
		}
	}

	return func() tea.Msg {
		result, err := cacheSvc.GetSleepForCycle(ctx, cycleID)
		if err != nil {
			return SleepMsg{Err: err}
		}
		if result == nil {
			return SleepMsg{Sleep: nil}
		}
		return SleepMsg{Sleep: result.Data, FromCache: result.FromCache}
	}
}

type RecoveryMsg struct {
	Recovery  *whoop.Recovery
	FromCache bool
	Err       error
}

// FetchRecoveryCmd fetches recovery data for a cycle via CacheService.
func FetchRecoveryCmd(ctx context.Context, cacheSvc cache.CacheService, cycleID int64) tea.Cmd {
	if cacheSvc == nil {
		return func() tea.Msg {
			return RecoveryMsg{Recovery: nil}
		}
	}

	return func() tea.Msg {
		result, err := cacheSvc.GetRecoveryForCycle(ctx, cycleID)
		if err != nil {
			return RecoveryMsg{Err: err}
		}
		if result == nil {
			return RecoveryMsg{Recovery: nil}
		}
		return RecoveryMsg{Recovery: result.Data, FromCache: result.FromCache}
	}
}

type WorkoutsMsg struct {
	Workouts  []whoop.Workout
	FromCache bool
	Err       error
}

// FetchWorkoutsForDateCmd fetches workouts between cycleStart and referenceDate via CacheService.
func FetchWorkoutsForDateCmd(ctx context.Context, cacheSvc cache.CacheService, cycleStart, referenceDate time.Time) tea.Cmd {
	if cacheSvc == nil {
		return func() tea.Msg {
			return WorkoutsMsg{}
		}
	}

	return func() tea.Msg {
		endOfDay := xtime.StartOfDay(referenceDate).Add(24 * time.Hour)
		result, err := cacheSvc.GetWorkoutsForDateRange(ctx, cycleStart, endOfDay)
		if err != nil {
			return WorkoutsMsg{Err: err}
		}
		if result == nil {
			return WorkoutsMsg{}
		}
		return WorkoutsMsg{Workouts: result.Data, FromCache: result.FromCache}
	}
}

type HistoricalDataMsg struct {
	Recoveries []whoop.Recovery
	Cycles     []whoop.Cycle
	Sleeps     []whoop.Sleep
	FromCache  bool
	Err        error
	ErrSource  string
}

// CalendarDataMsg contains recovery and cycle data for calendar month coloring.
type CalendarDataMsg struct {
	Recoveries []whoop.Recovery
	Cycles     []whoop.Cycle
	FromCache  bool
	Err        error
}

// FetchCalendarDataCmd fetches recovery and cycle data for a month via CacheService.
func FetchCalendarDataCmd(ctx context.Context, cacheSvc cache.CacheService, month time.Time) tea.Cmd {
	if cacheSvc == nil {
		return func() tea.Msg {
			return CalendarDataMsg{}
		}
	}

	return func() tea.Msg {
		result, err := cacheSvc.GetCalendarData(ctx, month)
		if err != nil {
			return CalendarDataMsg{Err: err}
		}
		if result == nil {
			return CalendarDataMsg{}
		}
		return CalendarDataMsg{Recoveries: result.Recoveries, Cycles: result.Cycles, FromCache: result.FromCache}
	}
}

// FetchHistoricalDataCmd fetches 50 days of historical data ending at today.
func FetchHistoricalDataCmd(ctx context.Context, cacheSvc cache.CacheService) tea.Cmd {
	return FetchHistoricalDataForDateCmd(ctx, cacheSvc, time.Now())
}

// FetchHistoricalDataForDateCmd fetches 50 days of historical data ending at referenceDate via CacheService.
func FetchHistoricalDataForDateCmd(ctx context.Context, cacheSvc cache.CacheService, referenceDate time.Time) tea.Cmd {
	if cacheSvc == nil {
		return func() tea.Msg {
			return HistoricalDataMsg{}
		}
	}

	return func() tea.Msg {
		result, err := cacheSvc.GetHistoricalData(ctx, referenceDate, 50)
		if err != nil {
			return HistoricalDataMsg{Err: err, ErrSource: "cache"}
		}
		if result == nil {
			return HistoricalDataMsg{}
		}
		return HistoricalDataMsg{
			Recoveries: result.Recoveries,
			Cycles:     result.Cycles,
			Sleeps:     result.Sleeps,
			FromCache:  result.FromCache,
		}
	}
}

func ComputeAverages(recoveries []whoop.Recovery, cycles []whoop.Cycle, sleeps []whoop.Sleep) *ThirtyDayAverages {
	if len(recoveries) == 0 && len(cycles) == 0 && len(sleeps) == 0 {
		return nil
	}

	var (
		hrvSum       float64
		rhrSum       float64
		respSum      float64
		sleepSum     float64
		strainSum    float64
		caloriesSum  float64
		avgHRSum     float64
		maxHRSum     float64
		hrvCount     int
		rhrCount     int
		respCount    int
		sleepCount   int
		strainCount  int
		calorieCount int
		avgHRCount   int
		maxHRCount   int
	)

	for _, r := range recoveries {
		if r.Score != nil {
			if r.Score.HRVRmssdMilli > 0 {
				hrvSum += r.Score.HRVRmssdMilli
				hrvCount++
			}
			if r.Score.RestingHeartRate > 0 {
				rhrSum += r.Score.RestingHeartRate
				rhrCount++
			}
		}
	}

	for _, c := range cycles {
		if c.Score != nil {
			if c.Score.Strain > 0 {
				strainSum += c.Score.Strain
				strainCount++
			}
			if c.Score.Kilojoule > 0 {
				caloriesSum += c.Score.Kilojoule / 4.184
				calorieCount++
			}
			if c.Score.AverageHeartRate > 0 {
				avgHRSum += float64(c.Score.AverageHeartRate)
				avgHRCount++
			}
			if c.Score.MaxHeartRate > 0 {
				maxHRSum += float64(c.Score.MaxHeartRate)
				maxHRCount++
			}
		}
	}

	for _, s := range sleeps {
		if s.Score != nil && !s.Nap {
			if s.Score.RespiratoryRate > 0 {
				respSum += s.Score.RespiratoryRate
				respCount++
			}
			if s.Score.SleepPerformancePercentage > 0 {
				sleepSum += s.Score.SleepPerformancePercentage
				sleepCount++
			}
		}
	}

	avg := &ThirtyDayAverages{}

	if hrvCount > 0 {
		avg.HRV = hrvSum / float64(hrvCount)
	}
	if rhrCount > 0 {
		avg.RestingHeartRate = rhrSum / float64(rhrCount)
	}
	if respCount > 0 {
		avg.RespiratoryRate = respSum / float64(respCount)
	}
	if sleepCount > 0 {
		avg.SleepPerformance = sleepSum / float64(sleepCount)
	}
	if strainCount > 0 {
		avg.Strain = strainSum / float64(strainCount)
	}
	if calorieCount > 0 {
		avg.Calories = caloriesSum / float64(calorieCount)
	}
	if avgHRCount > 0 {
		avg.AvgHeartRate = avgHRSum / float64(avgHRCount)
	}
	if maxHRCount > 0 {
		avg.MaxHeartRate = maxHRSum / float64(maxHRCount)
	}

	return avg
}
