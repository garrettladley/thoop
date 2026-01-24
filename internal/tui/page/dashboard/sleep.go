package dashboard

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/components/metricrow"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

func RenderSleepDetail(state State, width, height int) string {
	sleepGauge := gauge.New(
		state.SleepScore,
		100,
		"SLEEP PERFORMANCE",
		theme.ColorSleep,
	)

	gaugeStr := sleepGauge.Render()

	metricsWidth := 40
	metrics := renderSleepMetrics(state, metricsWidth)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		gaugeStr,
		"",
		"",
		metrics,
	)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func renderSleepMetrics(state State, width int) string {
	var rows []string

	if state.CurrentSleep != nil && state.CurrentSleep.Score != nil {
		score := state.CurrentSleep.Score

		hoursSlept := calculateHoursSlept(score.StageSummary.TotalInBedTimeMilli - score.StageSummary.TotalAwakeTimeMilli)
		hoursNeeded := calculateHoursNeeded(
			score.SleepNeeded.BaselineMilli,
			score.SleepNeeded.NeedFromSleepDebtMilli,
			score.SleepNeeded.NeedFromRecentStrainMilli,
			score.SleepNeeded.NeedFromRecentNapMilli,
		)
		sleepRatio := 0.0
		if hoursNeeded > 0 {
			sleepRatio = hoursSlept / hoursNeeded
		}
		sleepPct := sleepRatio * 100

		sleptRow := metricrow.New(
			"Hours vs. Needed",
			fmt.Sprintf("%.0f%%", sleepPct),
			width,
			metricrow.WithProgressBar(sleepRatio, metricrow.PercentageColor(sleepPct)),
		)
		rows = append(rows, sleptRow.Render())
		rows = append(rows, "")

		consistencyRow := metricrow.New(
			"Sleep Consistency",
			fmt.Sprintf("%.0f%%", score.SleepConsistencyPercentage),
			width,
			metricrow.WithProgressBar(score.SleepConsistencyPercentage/100, metricrow.PercentageColor(score.SleepConsistencyPercentage)),
		)
		rows = append(rows, consistencyRow.Render())
		rows = append(rows, "")

		efficiencyRow := metricrow.New(
			"Sleep Efficiency",
			fmt.Sprintf("%.0f%%", score.SleepEfficiencyPercentage),
			width,
			metricrow.WithProgressBar(score.SleepEfficiencyPercentage/100, metricrow.PercentageColor(score.SleepEfficiencyPercentage)),
		)
		rows = append(rows, efficiencyRow.Render())
		rows = append(rows, "")

		// padding to match recovery/strain page heights (they have 4 metrics with sub-values)
		rows = append(rows, "")
		rows = append(rows, "")
		rows = append(rows, "")

		rows = append(rows, "")
		rows = append(rows, renderSleepLegend())
	} else {
		noDataStyle := lipgloss.NewStyle().
			Foreground(theme.ColorDim).
			Italic(true)
		rows = append(rows, noDataStyle.Render("No sleep data available"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderSleepLegend() string {
	poorBox := lipgloss.NewStyle().Foreground(theme.ColorOrange).Render("■")
	poorText := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(" Poor  ")
	suffBox := lipgloss.NewStyle().Foreground(theme.ColorNeutral).Render("■")
	suffText := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(" Sufficient  ")
	optBox := lipgloss.NewStyle().Foreground(theme.ColorTeal).Render("■")
	optText := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(" Optimal")
	return poorBox + poorText + suffBox + suffText + optBox + optText
}

func calculateHoursSlept(totalSleepMilli int) float64 {
	return float64(totalSleepMilli) / (1000 * 60 * 60)
}

func calculateHoursNeeded(baselineMilli, debtMilli, strainMilli, napMilli int) float64 {
	totalMilli := baselineMilli + debtMilli + strainMilli - napMilli
	return float64(totalMilli) / (1000 * 60 * 60)
}
