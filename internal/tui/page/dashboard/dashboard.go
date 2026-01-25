package dashboard

import (
	"image/color"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/tui/components/auth"
	"github.com/garrettladley/thoop/internal/tui/components/date_header"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

type Tab int

const (
	TabOverview Tab = iota
	TabSleep
	TabRecovery
	TabStrain
)

func (t Tab) String() string {
	switch t {
	case TabOverview:
		return "Overview"
	case TabSleep:
		return "Sleep"
	case TabRecovery:
		return "Recovery"
	case TabStrain:
		return "Strain"
	default:
		return "Unknown"
	}
}

type ThirtyDayAverages struct {
	HRV              float64
	RestingHeartRate float64
	RespiratoryRate  float64
	SleepPerformance float64
	Strain           float64
	Calories         float64
	AvgHeartRate     float64
	MaxHeartRate     float64
}

type State struct {
	AuthIndicator auth.Indicator
	ActiveTab     Tab

	// selectedDate is the currently viewed date; nil means "today"
	SelectedDate *time.Time

	CycleID       int64
	SleepScore    *float64 // 0-100%
	RecoveryScore *float64 // 0-100%
	StrainScore   *float64 // 0-21

	CurrentSleep    *whoop.Sleep
	CurrentRecovery *whoop.Recovery
	CurrentCycle    *whoop.Cycle
	Averages        *ThirtyDayAverages

	// today's workouts for the strain page
	TodaysWorkouts []whoop.Workout

	// historical data for charts (last 7 days)
	HistoricalRecoveries []whoop.Recovery
	HistoricalCycles     []whoop.Cycle
	HistoricalSleeps     []whoop.Sleep

	// scrollOffset is the vertical scroll position for drill pages
	ScrollOffset int

	// loading tracks pending sleep/recovery requests after cycle fetch
	pendingSleep    bool
	pendingRecovery bool
}

// EffectiveDate returns the selected date, or today if none is selected.
func (s *State) EffectiveDate() time.Time {
	if s.SelectedDate != nil {
		return *s.SelectedDate
	}
	return time.Now()
}

func (s *State) Loading() bool {
	return s.pendingSleep || s.pendingRecovery
}

func (s *State) DataReady() bool {
	return s.CurrentCycle != nil && s.CurrentSleep != nil && s.CurrentRecovery != nil
}

func (s *State) SetPending() {
	s.pendingSleep = true
	s.pendingRecovery = true
}

func (s *State) ClearPendingSleep() {
	s.pendingSleep = false
}

func (s *State) ClearPendingRecovery() {
	s.pendingRecovery = false
}

func View(state *State, width, height int) string {
	dateHeader := date_header.Render(state.SelectedDate, width)
	dateHeaderHeight := lipgloss.Height(dateHeader)
	contentHeight := height - dateHeaderHeight

	var content string
	switch state.ActiveTab {
	case TabSleep:
		content = RenderSleepDetail(state, width, contentHeight)
	case TabRecovery:
		content = RenderRecoveryDetail(state, width, contentHeight)
	case TabStrain:
		content = RenderStrainDetail(state, width, contentHeight)
	default:
		content = renderOverview(*state, width, contentHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, dateHeader, content)
}

func renderOverview(state State, width, height int) string {
	var sleepScore, recoveryScore, strainScore *float64
	if state.DataReady() {
		sleepScore = state.SleepScore
		recoveryScore = state.RecoveryScore
		strainScore = state.StrainScore
	}

	var (
		sleepGauge = gauge.New(
			sleepScore,
			100,
			"SLEEP",
			theme.ColorSleep,
		)

		recoveryGauge = gauge.New(
			recoveryScore,
			100,
			"RECOVERY",
			recoveryColor(recoveryScore),
		)

		strainGauge = gauge.New(
			strainScore,
			21,
			"STRAIN",
			theme.ColorStrain,
		)
	)

	gaugeSpacing := "    "
	gaugesRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sleepGauge.Render(),
		gaugeSpacing,
		recoveryGauge.Render(),
		gaugeSpacing,
		strainGauge.Render(),
	)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		gaugesRow,
	)
}

func AuthIndicatorView(state State) string {
	return state.AuthIndicator.Render()
}

func recoveryColor(score *float64) color.Color {
	if score == nil {
		return theme.ColorRecoveryBlue
	}

	s := *score
	switch {
	case s >= 67:
		return theme.ColorHighRecovery
	case s >= 34:
		return theme.ColorMediumRecovery
	default:
		return theme.ColorLowRecovery
	}
}
