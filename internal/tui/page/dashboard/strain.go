package dashboard

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/components/metricrow"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

func RenderStrainDetail(state State, width, height int) string {
	strainGauge := gauge.New(
		state.StrainScore,
		21,
		"STRAIN",
		theme.ColorStrain,
	)

	gaugeStr := strainGauge.Render()

	metricsWidth := 40
	metrics := renderStrainMetrics(state, metricsWidth)

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

func renderStrainMetrics(state State, width int) string {
	var rows []string

	if state.CurrentCycle != nil && state.CurrentCycle.Score != nil {
		score := state.CurrentCycle.Score

		strainAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.Strain })
		strainDirection := getDirectionHigherBetter(score.Strain, strainAvg)
		strainRow := metricrow.New(
			"Strain",
			fmt.Sprintf("%.1f", score.Strain),
			width,
			metricrow.WithDirection(strainDirection),
			metricrow.WithSubValue(formatAvg(strainAvg, "%.1f")),
		)
		rows = append(rows, strainRow.Render())
		rows = append(rows, "")

		const kjPerKcal = 4.184
		calories := score.Kilojoule / kjPerKcal
		caloriesAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.Calories })
		caloriesDirection := getDirectionHigherBetter(calories, caloriesAvg)
		caloriesRow := metricrow.New(
			"Calories",
			fmt.Sprintf("%.0f", calories),
			width,
			metricrow.WithDirection(caloriesDirection),
			metricrow.WithSubValue(formatAvg(caloriesAvg, "%.0f")),
		)
		rows = append(rows, caloriesRow.Render())
		rows = append(rows, "")

		avgHR := float64(score.AverageHeartRate)
		avgHRAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.AvgHeartRate })
		avgHRDirection := getDirectionHigherBetter(avgHR, avgHRAvg)
		avgHRRow := metricrow.New(
			"Avg Heart Rate",
			fmt.Sprintf("%.0f", avgHR),
			width,
			metricrow.WithDirection(avgHRDirection),
			metricrow.WithSubValue(formatAvg(avgHRAvg, "%.0f")),
		)
		rows = append(rows, avgHRRow.Render())
		rows = append(rows, "")

		maxHR := float64(score.MaxHeartRate)
		maxHRAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.MaxHeartRate })
		maxHRDirection := getDirectionHigherBetter(maxHR, maxHRAvg)
		maxHRRow := metricrow.New(
			"Max Heart Rate",
			fmt.Sprintf("%.0f", maxHR),
			width,
			metricrow.WithDirection(maxHRDirection),
			metricrow.WithSubValue(formatAvg(maxHRAvg, "%.0f")),
		)
		rows = append(rows, maxHRRow.Render())
		rows = append(rows, "")

		rows = append(rows, "")
		rows = append(rows, renderComparisonLegend())
	} else {
		noDataStyle := lipgloss.NewStyle().
			Foreground(theme.ColorDim).
			Italic(true)
		rows = append(rows, noDataStyle.Render("No strain data available"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
