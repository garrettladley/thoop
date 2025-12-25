package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/auth"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

type DashboardState struct {
	AuthIndicator auth.Indicator

	CycleID       int64
	SleepScore    *float64 // 0-100%
	RecoveryScore *float64 // 0-100%
	StrainScore   *float64 // 0-21
}

func (m *Model) DashboardView() string {
	var (
		sleepGauge = gauge.New(
			m.state.dashboard.SleepScore,
			100,
			"SLEEP",
			theme.ColorSleep,
		)

		recoveryGauge = gauge.New(
			m.state.dashboard.RecoveryScore,
			100,
			"RECOVERY",
			m.recoveryColor(),
		)

		strainGauge = gauge.New(
			m.state.dashboard.StrainScore,
			21,
			"STRAIN",
			theme.ColorStrain,
		)
	)

	// render gauges side by side with spacing
	gaugeSpacing := "    "
	gaugesRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sleepGauge.Render(),
		gaugeSpacing,
		recoveryGauge.Render(),
		gaugeSpacing,
		strainGauge.Render(),
	)

	return gaugesRow
}

func (m *Model) AuthIndicatorView() string {
	return m.state.dashboard.AuthIndicator.Render()
}

func (m *Model) recoveryColor() color.Color {
	if m.state.dashboard.RecoveryScore == nil {
		return theme.ColorRecoveryBlue // neutral color when no data
	}

	score := *m.state.dashboard.RecoveryScore
	switch {
	case score >= 67:
		return theme.ColorHighRecovery
	case score >= 34:
		return theme.ColorMediumRecovery
	default:
		return theme.ColorLowRecovery
	}
}
