package dashboard

import (
	"context"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/xtime"
)

// DateChangedMsg signals that the selected date has changed.
type DateChangedMsg struct {
	Date *time.Time
}

type CycleMsg struct {
	Cycle *whoop.Cycle
	Err   error
}

// FetchCycleCmd fetches the most recent cycle (today's cycle).
func FetchCycleCmd(ctx context.Context, client *whoop.Client) tea.Cmd {
	return FetchCycleForDateCmd(ctx, client, time.Now())
}

// FetchCycleForDateCmd fetches the cycle containing the given reference date.
func FetchCycleForDateCmd(ctx context.Context, client *whoop.Client, referenceDate time.Time) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return CycleMsg{Cycle: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// query cycles that span the reference date
		start := xtime.StartOfDay(referenceDate)
		end := start.Add(24 * time.Hour)

		cycles, err := client.Cycle.List(ctx, &whoop.ListParams{
			Limit: 1,
			Start: &start,
			End:   &end,
		})
		if err != nil {
			return CycleMsg{Err: err}
		}
		if len(cycles.Records) == 0 {
			return CycleMsg{Cycle: nil}
		}
		return CycleMsg{Cycle: &cycles.Records[0]}
	}
}

type SleepMsg struct {
	Sleep *whoop.Sleep
	Err   error
}

func FetchSleepCmd(ctx context.Context, client *whoop.Client, cycleID int64) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return SleepMsg{Sleep: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		sleep, err := client.Cycle.GetSleep(ctx, cycleID)
		return SleepMsg{Sleep: sleep, Err: err}
	}
}

type RecoveryMsg struct {
	Recovery *whoop.Recovery
	Err      error
}

func FetchRecoveryCmd(ctx context.Context, client *whoop.Client, cycleID int64) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return RecoveryMsg{Recovery: nil}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		recovery, err := client.Cycle.GetRecovery(ctx, cycleID)
		return RecoveryMsg{Recovery: recovery, Err: err}
	}
}

type WorkoutsMsg struct {
	Workouts []whoop.Workout
	Err      error
}

// FetchTodaysWorkoutsCmd fetches workouts for today (wrapper for FetchWorkoutsForDateCmd).
func FetchTodaysWorkoutsCmd(ctx context.Context, client *whoop.Client, cycleStart time.Time) tea.Cmd {
	return FetchWorkoutsForDateCmd(ctx, client, cycleStart, time.Now())
}

// FetchWorkoutsForDateCmd fetches workouts between cycleStart and referenceDate.
func FetchWorkoutsForDateCmd(ctx context.Context, client *whoop.Client, cycleStart, referenceDate time.Time) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return WorkoutsMsg{}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		endOfDay := xtime.StartOfDay(referenceDate).Add(24 * time.Hour)
		resp, err := client.Workout.List(ctx, &whoop.ListParams{
			Start: &cycleStart,
			End:   &endOfDay,
		})
		if err != nil {
			return WorkoutsMsg{Err: err}
		}
		return WorkoutsMsg{Workouts: resp.Records}
	}
}

type HistoricalDataMsg struct {
	Recoveries []whoop.Recovery
	Cycles     []whoop.Cycle
	Sleeps     []whoop.Sleep
	Err        error
	ErrSource  string
}

// FetchHistoricalDataCmd fetches 30 days of historical data ending at today.
func FetchHistoricalDataCmd(ctx context.Context, client *whoop.Client) tea.Cmd {
	return FetchHistoricalDataForDateCmd(ctx, client, time.Now())
}

// FetchHistoricalDataForDateCmd fetches 30 days of historical data ending at referenceDate.
func FetchHistoricalDataForDateCmd(ctx context.Context, client *whoop.Client, referenceDate time.Time) tea.Cmd {
	if client == nil {
		return func() tea.Msg {
			return HistoricalDataMsg{}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		endOfDay := xtime.StartOfDay(referenceDate).Add(24 * time.Hour)
		thirtyDaysAgo := endOfDay.AddDate(0, 0, -30)

		var (
			allRecoveries []whoop.Recovery
			allCycles     []whoop.Cycle
			allSleeps     []whoop.Sleep
			recoveryErr   error
			cycleErr      error
			sleepErr      error
			wg            sync.WaitGroup
		)

		wg.Go(func() {
			var nextToken *string
			for {
				params := &whoop.ListParams{
					Limit:     25,
					Start:     &thirtyDaysAgo,
					End:       &endOfDay,
					NextToken: nextToken,
				}
				resp, err := client.Cycle.List(ctx, params)
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
		})

		wg.Go(func() {
			var nextToken *string
			for {
				params := &whoop.ListParams{
					Limit:     25,
					Start:     &thirtyDaysAgo,
					End:       &endOfDay,
					NextToken: nextToken,
				}
				resp, err := client.Sleep.List(ctx, params)
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
		})

		var nextToken *string
		for {
			params := &whoop.ListParams{
				Limit:     25,
				Start:     &thirtyDaysAgo,
				End:       &endOfDay,
				NextToken: nextToken,
			}
			resp, err := client.Recovery.List(ctx, params)
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

		if recoveryErr != nil {
			return HistoricalDataMsg{Err: recoveryErr, ErrSource: "recovery"}
		}
		if cycleErr != nil {
			return HistoricalDataMsg{Err: cycleErr, ErrSource: "cycle"}
		}
		if sleepErr != nil {
			return HistoricalDataMsg{Err: sleepErr, ErrSource: "sleep"}
		}

		return HistoricalDataMsg{
			Recoveries: allRecoveries,
			Cycles:     allCycles,
			Sleeps:     allSleeps,
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
