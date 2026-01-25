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

	var contentParts []string
	contentParts = append(contentParts, gaugeStr)
	contentParts = append(contentParts, "", "")
	contentParts = append(contentParts, metrics)

	chartsSection := renderSleepCharts(state, metricsWidth)
	if chartsSection != "" {
		contentParts = append(contentParts, "", "", "")
		contentParts = append(contentParts, chartsSection)
	}

	content := lipgloss.JoinVertical(lipgloss.Center, contentParts...)

	contentHeight := lipgloss.Height(content)
	viewportHeight := height - 2 // reserve space for footer

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

		sleptRow := metric_row.New(
			"Hours vs. Needed",
			fmt.Sprintf("%.0f%%", sleepPct),
			width,
			metric_row.WithProgressBar(sleepRatio, metric_row.PercentageColor(sleepPct)),
		)
		rows = append(rows, sleptRow.Render())
		rows = append(rows, "")

		consistencyRow := metric_row.New(
			"Sleep Consistency",
			fmt.Sprintf("%.0f%%", score.SleepConsistencyPercentage),
			width,
			metric_row.WithProgressBar(score.SleepConsistencyPercentage/100, metric_row.PercentageColor(score.SleepConsistencyPercentage)),
		)
		rows = append(rows, consistencyRow.Render())
		rows = append(rows, "")

		efficiencyRow := metric_row.New(
			"Sleep Efficiency",
			fmt.Sprintf("%.0f%%", score.SleepEfficiencyPercentage),
			width,
			metric_row.WithProgressBar(score.SleepEfficiencyPercentage/100, metric_row.PercentageColor(score.SleepEfficiencyPercentage)),
		)
		rows = append(rows, efficiencyRow.Render())
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

func renderSleepCharts(state State, width int) string {
	if len(state.HistoricalSleeps) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true)

	var sections []string
	sections = append(sections, titleStyle.Render("WEEKLY SLEEP PERFORMANCE"))
	sections = append(sections, "")

	var perfData []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(perfData) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		perfData = append(perfData, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepPerformancePercentage,
		})
	}

	for i, j := 0, len(perfData)-1; i < j; i, j = i+1, j-1 {
		perfData[i], perfData[j] = perfData[j], perfData[i]
	}

	if len(perfData) > 0 {
		perfChart := chart.NewBarChart(perfData,
			chart.WithBarChartColorFunc(chart.SleepPerformanceColor),
			chart.WithBarChartFormatter(chart.FormatPercentage),
			chart.WithBarChartMax(100),
			chart.WithBarChartHeight(6),
			chart.WithBarChartShowValues(true),
		)
		sections = append(sections, perfChart.Render(width))
	}

	// add hours slept vs needed dual line chart
	sections = append(sections, "")
	sections = append(sections, "")
	sections = append(sections, titleStyle.Render("HOURS OF SLEEP VS NEEDED"))
	sections = append(sections, "")

	var actualData, neededData []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(actualData) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		hoursSlept := calculateHoursSlept(s.Score.StageSummary.TotalInBedTimeMilli - s.Score.StageSummary.TotalAwakeTimeMilli)
		hoursNeeded := calculateHoursNeeded(
			s.Score.SleepNeeded.BaselineMilli,
			s.Score.SleepNeeded.NeedFromSleepDebtMilli,
			s.Score.SleepNeeded.NeedFromRecentStrainMilli,
			s.Score.SleepNeeded.NeedFromRecentNapMilli,
		)
		label := s.CreatedAt.Format("Mon\n2")
		actualData = append(actualData, chart.DataPoint{Label: label, Value: hoursSlept})
		neededData = append(neededData, chart.DataPoint{Label: label, Value: hoursNeeded})
	}

	for i, j := 0, len(actualData)-1; i < j; i, j = i+1, j-1 {
		actualData[i], actualData[j] = actualData[j], actualData[i]
		neededData[i], neededData[j] = neededData[j], neededData[i]
	}

	if len(actualData) > 0 {
		dualChart := chart.NewDualLineChart(actualData, neededData,
			chart.WithDualLineColors(theme.ColorSleep, theme.ColorDim),
			chart.WithDualLineLabels("Actual", "Needed"),
			chart.WithDualLineHeight(5),
			chart.WithDualLineMin(0),
			chart.WithDualLineMax(12),
			chart.WithDualLineShowLegend(true),
		)
		sections = append(sections, dualChart.Render(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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
