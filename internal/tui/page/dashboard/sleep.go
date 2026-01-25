package dashboard

import (
	"fmt"
	colorpkg "image/color"

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

	metricsWidth := 56
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

		sleepPerfRow := metric_row.New(
			"Sleep Performance",
			fmt.Sprintf("%.0f%%", score.SleepPerformancePercentage),
			width,
			metric_row.WithSegmentedProgressBarThresholds(score.SleepPerformancePercentage, 3, 70, 85),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, sleepPerfRow.Render())
		rows = append(rows, "")

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
			metric_row.WithSegmentedProgressBarThresholds(sleepPct, 3, 70, 85),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, sleptRow.Render())
		rows = append(rows, "")

		consistencyRow := metric_row.New(
			"Sleep Consistency",
			fmt.Sprintf("%.0f%%", score.SleepConsistencyPercentage),
			width,
			metric_row.WithSegmentedProgressBarThresholds(score.SleepConsistencyPercentage, 3, 70, 80),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, consistencyRow.Render())
		rows = append(rows, "")

		efficiencyRow := metric_row.New(
			"Sleep Efficiency",
			fmt.Sprintf("%.0f%%", score.SleepEfficiencyPercentage),
			width,
			metric_row.WithSegmentedProgressBarThresholds(score.SleepEfficiencyPercentage, 3, 80, 90),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, efficiencyRow.Render())
		rows = append(rows, "")

		rows = append(rows, "")
		rows = append(rows, renderSleepLegend())

		rows = append(rows, "", "")
		rows = append(rows, renderSleepStagesChart(state, width))

		rows = append(rows, "")
		rows = append(rows, renderRestorativeSleepFooter(state, width))
	} else {
		noDataStyle := lipgloss.NewStyle().
			Foreground(theme.ColorDim).
			Italic(true)
		rows = append(rows, noDataStyle.Render("No sleep data available"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderSleepStagesChart(state State, width int) string {
	if state.CurrentSleep == nil || state.CurrentSleep.Score == nil {
		return ""
	}

	score := state.CurrentSleep.Score
	stages := score.StageSummary

	// calculate total sleep time (excluding awake)
	totalSleepMs := stages.TotalLightSleepTimeMilli +
		stages.TotalSlowWaveSleepTimeMilli +
		stages.TotalREMSleepTimeMilli

	if totalSleepMs <= 0 {
		return ""
	}

	// calculate percentages
	awakePct := float64(stages.TotalAwakeTimeMilli) / float64(totalSleepMs+stages.TotalAwakeTimeMilli) * 100
	lightPct := float64(stages.TotalLightSleepTimeMilli) / float64(totalSleepMs) * 100
	deepPct := float64(stages.TotalSlowWaveSleepTimeMilli) / float64(totalSleepMs) * 100
	remPct := float64(stages.TotalREMSleepTimeMilli) / float64(totalSleepMs) * 100

	// Total duration for display (in bed time minus awake)
	totalDuration := totalSleepMs

	stageData := []chart.SleepStage{
		{
			Name:       "AWAKE",
			DurationMs: stages.TotalAwakeTimeMilli,
			Percentage: awakePct,
			Color:      chart.ColorSleepAwake,
		},
		{
			Name:       "LIGHT",
			DurationMs: stages.TotalLightSleepTimeMilli,
			Percentage: lightPct,
			Color:      chart.ColorSleepLight,
		},
		{
			Name:       "SWS (DEEP)",
			DurationMs: stages.TotalSlowWaveSleepTimeMilli,
			Percentage: deepPct,
			Color:      chart.ColorSleepDeep,
		},
		{
			Name:       "REM",
			DurationMs: stages.TotalREMSleepTimeMilli,
			Percentage: remPct,
			Color:      chart.ColorSleepREM,
		},
	}

	// calculate baseline sleep duration from historical data
	baselineSleepMs := calculateSleepDurationBaseline(state)

	stagesChart := chart.NewSleepStagesChart(stageData, totalDuration,
		chart.WithSleepStagesBaseline(baselineSleepMs),
	)
	return stagesChart.Render(width)
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

	if len(actualData) > 0 {
		dualChart := chart.NewDualLineChart(actualData, neededData,
			chart.WithDualLineColors(theme.ColorSleep, theme.ColorTeal),
			chart.WithDualLineLabels("HOURS OF SLEEP", "SLEEP NEEDED"),
			chart.WithDualLineHeight(5),
			chart.WithDualLineAutoScale(true),
			chart.WithDualLineShowValues(true),
			chart.WithDualLineShowLegend(true),
			chart.WithDualLineLegendPosition(chart.LegendTopRight),
			chart.WithDualLineFormatter(chart.FormatDurationFromHours),
		)
		sections = append(sections, dualChart.Render(width))
	}

	sections = append(sections, "", "")
	sections = append(sections, titleStyle.Render("HOURS VS NEEDED (%)"))
	sections = append(sections, "")

	var hoursVsNeededData []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(hoursVsNeededData) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		hoursVsNeededData = append(hoursVsNeededData, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepPerformancePercentage,
		})
	}

	if len(hoursVsNeededData) > 0 {
		hoursVsNeededChart := chart.NewBarChart(hoursVsNeededData,
			chart.WithBarChartColorFunc(chart.SleepColor),
			chart.WithBarChartFormatter(chart.FormatPercentage),
			chart.WithBarChartMax(100),
			chart.WithBarChartHeight(6),
			chart.WithBarChartShowValues(true),
		)
		sections = append(sections, hoursVsNeededChart.Render(width))
	}

	sections = append(sections, "", "")
	sections = append(sections, titleStyle.Render("RESTORATIVE SLEEP"))
	sections = append(sections, "")

	var restorativeData []chart.StackedDataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(restorativeData) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		stages := s.Score.StageSummary
		remHours := float64(stages.TotalREMSleepTimeMilli) / (1000 * 60 * 60)
		deepHours := float64(stages.TotalSlowWaveSleepTimeMilli) / (1000 * 60 * 60)
		restorativeData = append(restorativeData, chart.StackedDataPoint{
			Label:  s.CreatedAt.Format("Mon\n2"),
			Values: []float64{remHours, deepHours},
		})
	}

	if len(restorativeData) > 0 {
		restorativeChart := chart.NewStackedBarChart(restorativeData,
			chart.WithStackedBarColors([]colorpkg.Color{chart.ColorSleepDeep, chart.ColorSleepREM}),
			chart.WithStackedBarLabels([]string{"DEEP SLEEP", "REM SLEEP"}),
			chart.WithStackedBarFormatter(chart.FormatDurationFromHours),
			chart.WithStackedBarHeight(6),
			chart.WithStackedBarShowLegend(true),
			chart.WithStackedBarLegendPosition(chart.LegendTopRight),
		)
		sections = append(sections, restorativeChart.Render(width))
	}

	sections = append(sections, "", "")
	sections = append(sections, titleStyle.Render("SLEEP CONSISTENCY (%)"))
	sections = append(sections, "")

	var consistencyData []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(consistencyData) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		consistencyData = append(consistencyData, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepConsistencyPercentage,
		})
	}

	if len(consistencyData) > 0 {
		consistencyChart := chart.NewBarChart(consistencyData,
			chart.WithBarChartColorFunc(chart.SleepColor),
			chart.WithBarChartFormatter(chart.FormatPercentage),
			chart.WithBarChartMax(100),
			chart.WithBarChartHeight(6),
			chart.WithBarChartShowValues(true),
		)
		sections = append(sections, consistencyChart.Render(width))
	}

	sections = append(sections, "", "")
	sections = append(sections, titleStyle.Render("SLEEP EFFICIENCY (%)"))
	sections = append(sections, "")

	var efficiencyData []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(efficiencyData) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		efficiencyData = append(efficiencyData, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepEfficiencyPercentage,
		})
	}

	if len(efficiencyData) > 0 {
		efficiencyChart := chart.NewLineChart(efficiencyData,
			chart.WithLineChartColor(theme.ColorSleep),
			chart.WithLineChartFormatter(chart.FormatPercentage),
			chart.WithLineChartHeight(5),
			chart.WithLineChartShowValues(true),
			chart.WithLineChartShowDots(true),
		)
		sections = append(sections, efficiencyChart.Render(width))
	}

	sections = append(sections, "", "")
	sections = append(sections, titleStyle.Render("SLEEP DEBT"))
	sections = append(sections, "")

	var debtData []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(debtData) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		debtHours := float64(s.Score.SleepNeeded.NeedFromSleepDebtMilli) / (1000 * 60 * 60)
		debtData = append(debtData, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: debtHours,
		})
	}

	if len(debtData) > 0 {
		debtChart := chart.NewBarChart(debtData,
			chart.WithBarChartColorFunc(chart.SleepColor),
			chart.WithBarChartFormatter(chart.FormatDurationFromHours),
			chart.WithBarChartMax(3),
			chart.WithBarChartHeight(6),
			chart.WithBarChartShowValues(true),
		)
		sections = append(sections, debtChart.Render(width))
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

func renderRestorativeSleepFooter(state State, width int) string {
	if state.CurrentSleep == nil || state.CurrentSleep.Score == nil {
		return ""
	}

	score := state.CurrentSleep.Score
	stages := score.StageSummary

	currentRestorativeMs := stages.TotalREMSleepTimeMilli + stages.TotalSlowWaveSleepTimeMilli

	baselineRestorativeMs := calculateRestorativeBaseline(state)

	splitSquare := renderSplitSquare()

	labelStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite).Bold(true)
	label := labelStyle.Render("RESTORATIVE SLEEP")

	leftPart := splitSquare + " " + label

	currentDuration := formatDurationMs(currentRestorativeMs)
	valueStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite).Bold(true)

	var directionStr string
	if baselineRestorativeMs > 0 {
		if currentRestorativeMs > baselineRestorativeMs {
			directionStr = " " + lipgloss.NewStyle().Foreground(theme.ColorTeal).Render("▲")
		} else if currentRestorativeMs < baselineRestorativeMs {
			directionStr = " " + lipgloss.NewStyle().Foreground(theme.ColorOrange).Render("▼")
		}
	}

	valueText := valueStyle.Render(currentDuration) + directionStr

	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(valueText)
	padding := max(width-leftWidth-rightWidth, 1)

	firstLine := leftPart + fmt.Sprintf("%*s", padding, "") + valueText

	var lines []string
	lines = append(lines, firstLine)

	if baselineRestorativeMs > 0 {
		baselineDuration := formatDurationMs(baselineRestorativeMs)
		subValueStyle := lipgloss.NewStyle().Foreground(theme.ColorDim)
		subValueText := subValueStyle.Render(baselineDuration)
		subPadding := width - lipgloss.Width(subValueText)
		subLine := fmt.Sprintf("%*s", subPadding, "") + subValueText
		lines = append(lines, subLine)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderSplitSquare() string {
	// create a diagonal split square: top-left REM color, bottom-right SWS color
	// using ◤ (upper-left triangle) with foreground/background colors
	// the triangle (◤) is colored REM, the background (lower-right) is SWS
	style := lipgloss.NewStyle().
		Foreground(chart.ColorSleepREM).
		Background(chart.ColorSleepDeep)
	return style.Render("◤")
}

func calculateSleepDurationBaseline(state State) int {
	if len(state.HistoricalSleeps) == 0 {
		return 0
	}

	var totalMs int
	var count int

	for _, s := range state.HistoricalSleeps {
		if s.Nap || s.Score == nil {
			continue
		}
		stages := s.Score.StageSummary
		sleepMs := stages.TotalLightSleepTimeMilli +
			stages.TotalSlowWaveSleepTimeMilli +
			stages.TotalREMSleepTimeMilli
		totalMs += sleepMs
		count++
	}

	if count == 0 {
		return 0
	}

	return totalMs / count
}

func calculateRestorativeBaseline(state State) int {
	if len(state.HistoricalSleeps) == 0 {
		return 0
	}

	var totalMs int
	var count int

	for _, s := range state.HistoricalSleeps {
		if s.Nap || s.Score == nil {
			continue
		}
		stages := s.Score.StageSummary
		totalMs += stages.TotalREMSleepTimeMilli + stages.TotalSlowWaveSleepTimeMilli
		count++
	}

	if count == 0 {
		return 0
	}

	return totalMs / count
}

func formatDurationMs(ms int) string {
	totalMinutes := ms / (1000 * 60)
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	return fmt.Sprintf("%d:%02d", hours, minutes)
}
