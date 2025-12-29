package dashboard

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/auth"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

type State struct {
	AuthIndicator auth.Indicator

	CycleID       int64
	SleepScore    *float64 // 0-100%
	RecoveryScore *float64 // 0-100%
	StrainScore   *float64 // 0-21
}

func View(state State, width, height int) string {
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
