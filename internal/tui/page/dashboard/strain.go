package dashboard

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/chart"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/components/metric_row"
	"github.com/garrettladley/thoop/internal/tui/components/viewport"
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

	var contentParts []string
	contentParts = append(contentParts, gaugeStr)
	contentParts = append(contentParts, "", "")
	contentParts = append(contentParts, metrics)

	chartsSection := renderStrainCharts(state, metricsWidth)
	if chartsSection != "" {
		contentParts = append(contentParts, "", "", "")
		contentParts = append(contentParts, chartsSection)
	}

	content := lipgloss.JoinVertical(lipgloss.Center, contentParts...)

	contentHeight := lipgloss.Height(content)
	viewportHeight := height - 2

	if contentHeight <= viewportHeight {
		return lipgloss.Place(
			width,
			height,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)
	}

	vp := viewport.New(viewport.WithSize(width, viewportHeight))
	offset := vp.ClampOffset(content, state.ScrollOffset)
	scrolledContent := vp.Render(content, offset)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Top,
		scrolledContent,
	)
}

func renderStrainMetrics(state State, width int) string {
	var rows []string

	if state.CurrentCycle != nil && state.CurrentCycle.Score != nil {
		score := state.CurrentCycle.Score

		strainAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.Strain })
		strainDirection := getDirectionHigherBetter(score.Strain, strainAvg)
		strainRow := metric_row.New(
			"Strain",
			fmt.Sprintf("%.1f", score.Strain),
			width,
			metric_row.WithDirection(strainDirection),
			metric_row.WithSubValue(formatAvg(strainAvg, "%.1f")),
		)
		rows = append(rows, strainRow.Render())
		rows = append(rows, "")

		const kjPerKcal = 4.184
		calories := score.Kilojoule / kjPerKcal
		caloriesAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.Calories })
		caloriesDirection := getDirectionHigherBetter(calories, caloriesAvg)
		caloriesRow := metric_row.New(
			"Calories",
			fmt.Sprintf("%.0f", calories),
			width,
			metric_row.WithDirection(caloriesDirection),
			metric_row.WithSubValue(formatAvg(caloriesAvg, "%.0f")),
		)
		rows = append(rows, caloriesRow.Render())
		rows = append(rows, "")

		avgHR := float64(score.AverageHeartRate)
		avgHRAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.AvgHeartRate })
		avgHRDirection := getDirectionHigherBetter(avgHR, avgHRAvg)
		avgHRRow := metric_row.New(
			"Avg Heart Rate",
			fmt.Sprintf("%.0f", avgHR),
			width,
			metric_row.WithDirection(avgHRDirection),
			metric_row.WithSubValue(formatAvg(avgHRAvg, "%.0f")),
		)
		rows = append(rows, avgHRRow.Render())
		rows = append(rows, "")

		maxHR := float64(score.MaxHeartRate)
		maxHRAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.MaxHeartRate })
		maxHRDirection := getDirectionHigherBetter(maxHR, maxHRAvg)
		maxHRRow := metric_row.New(
			"Max Heart Rate",
			fmt.Sprintf("%.0f", maxHR),
			width,
			metric_row.WithDirection(maxHRDirection),
			metric_row.WithSubValue(formatAvg(maxHRAvg, "%.0f")),
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

func renderStrainCharts(state State, width int) string {
	if len(state.HistoricalCycles) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true)

	var sections []string
	sections = append(sections, titleStyle.Render("WEEKLY STRAIN"))
	sections = append(sections, "")

	// Build strain bar chart data
	var strainData []chart.DataPoint
	for i := len(state.HistoricalCycles) - 1; i >= 0 && len(strainData) < 7; i-- {
		c := state.HistoricalCycles[i]
		if c.Score == nil {
			continue
		}
		strainData = append(strainData, chart.DataPoint{
			Label: c.Start.Format("Mon\n2"),
			Value: c.Score.Strain,
		})
	}

	for i, j := 0, len(strainData)-1; i < j; i, j = i+1, j-1 {
		strainData[i], strainData[j] = strainData[j], strainData[i]
	}

	if len(strainData) > 0 {
		strainChart := chart.NewBarChart(strainData,
			chart.WithBarChartColorFunc(chart.StrainColor),
			chart.WithBarChartFormatter(chart.FormatFloat1),
			chart.WithBarChartMax(21),
			chart.WithBarChartHeight(6),
			chart.WithBarChartShowValues(true),
		)
		sections = append(sections, strainChart.Render(width))
	}

	sections = append(sections, "")
	sections = append(sections, "")
	sections = append(sections, titleStyle.Render("CALORIES BURNED"))
	sections = append(sections, "")

	var caloriesData []chart.DataPoint
	const kjPerKcal = 4.184
	for i := len(state.HistoricalCycles) - 1; i >= 0 && len(caloriesData) < 7; i-- {
		c := state.HistoricalCycles[i]
		if c.Score == nil {
			continue
		}
		caloriesData = append(caloriesData, chart.DataPoint{
			Label: c.Start.Format("Mon\n2"),
			Value: c.Score.Kilojoule / kjPerKcal,
		})
	}

	for i, j := 0, len(caloriesData)-1; i < j; i, j = i+1, j-1 {
		caloriesData[i], caloriesData[j] = caloriesData[j], caloriesData[i]
	}

	if len(caloriesData) > 0 {
		caloriesChart := chart.NewLineChart(caloriesData,
			chart.WithLineChartColor(theme.ColorStrain),
			chart.WithLineChartFormatter(chart.FormatInt),
			chart.WithLineChartHeight(5),
			chart.WithLineChartShowValues(true),
			chart.WithLineChartShowDots(true),
		)
		sections = append(sections, caloriesChart.Render(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
