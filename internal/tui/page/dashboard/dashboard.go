package dashboard

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/tui/components/auth"
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

	CycleID       int64
	SleepScore    *float64 // 0-100%
	RecoveryScore *float64 // 0-100%
	StrainScore   *float64 // 0-21

	CurrentSleep    *whoop.Sleep
	CurrentRecovery *whoop.Recovery
	CurrentCycle    *whoop.Cycle
	Averages        *ThirtyDayAverages

	// loading tracks pending sleep/recovery requests after cycle fetch
	pendingSleep    bool
	pendingRecovery bool
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

func View(state State, width, height int) string {
	switch state.ActiveTab {
	case TabSleep:
		return RenderSleepDetail(state, width, height)
	case TabRecovery:
		return RenderRecoveryDetail(state, width, height)
	case TabStrain:
		return RenderStrainDetail(state, width, height)
	default:
		return renderOverview(state, width, height)
	}
}

func renderOverview(state State, width, height int) string {
	var (
		sleepGauge = gauge.New(
			state.SleepScore,
			100,
			"SLEEP",
			theme.ColorSleep,
		)

		recoveryGauge = gauge.New(
			state.RecoveryScore,
			100,
			"RECOVERY",
			recoveryColor(state.RecoveryScore),
		)

		strainGauge = gauge.New(
			state.StrainScore,
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
