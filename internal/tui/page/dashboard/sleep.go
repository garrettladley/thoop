package dashboard

import (
	"fmt"
	colorpkg "image/color"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/chart"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/components/lazy_list"
	"github.com/garrettladley/thoop/internal/tui/components/metric_row"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

func RenderSleepDetail(state *State, width, height int) string {
	metricsWidth := 56
	viewportHeight := height - 2 // reserve space for footer

	items := buildSleepItems(state, metricsWidth)

	list := lazy_list.New(items)
	list.SetSize(metricsWidth, viewportHeight)

	totalHeight := list.TotalHeight()
	if totalHeight <= viewportHeight {
		state.ScrollOffset = 0
		content := list.Render()
		return lipgloss.Place(
			width,
			height,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)
	}

	offset := list.ClampOffset(state.ScrollOffset)
	state.ScrollOffset = offset
	list.SetOffset(offset)
	scrolledContent := list.Render()

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Top,
		scrolledContent,
	)
}

func buildSleepItems(state *State, metricsWidth int) []lazy_list.Item {
	items := make([]lazy_list.Item, 0, 20)

	sleepGauge := gauge.New(
		state.SleepScore,
		100,
		"SLEEP PERFORMANCE",
		theme.ColorSleep,
	)
	items = append(items, lazy_list.NewStaticItem(sleepGauge.Render()))

	items = append(items, lazy_list.NewSpacerItem(2))

	metrics := renderSleepMetrics(*state, metricsWidth)
	items = append(items, lazy_list.NewStaticItem(metrics))

	if len(state.HistoricalSleeps) > 0 {
		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("sleep-perf", func(w int) string {
			return sleepPerformanceChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("sleep-hours-dual", func(w int) string {
			return hoursVsNeededDualChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("sleep-hours-pct", func(w int) string {
			return hoursVsNeededPercentChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("sleep-restorative", func(w int) string {
			return restorativeSleepChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("sleep-consistency", func(w int) string {
			return sleepConsistencyChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("sleep-efficiency", func(w int) string {
			return sleepEfficiencyChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("sleep-debt", func(w int) string {
			return sleepDebtChart(*state, w)
		}))
	}

	return items
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

	awakePct := float64(stages.TotalAwakeTimeMilli) / float64(totalSleepMs+stages.TotalAwakeTimeMilli) * 100
	lightPct := float64(stages.TotalLightSleepTimeMilli) / float64(totalSleepMs) * 100
	deepPct := float64(stages.TotalSlowWaveSleepTimeMilli) / float64(totalSleepMs) * 100
	remPct := float64(stages.TotalREMSleepTimeMilli) / float64(totalSleepMs) * 100

	totalDuration := totalSleepMs

	stageData := []chart.SleepStage{
		{
			Name:       "AWAKE",
			DurationMs: stages.TotalAwakeTimeMilli,
			Percentage: awakePct,
			Color:      theme.ColorSleepAwake,
		},
		{
			Name:       "LIGHT",
			DurationMs: stages.TotalLightSleepTimeMilli,
			Percentage: lightPct,
			Color:      theme.ColorSleepLight,
		},
		{
			Name:       "SWS (DEEP)",
			DurationMs: stages.TotalSlowWaveSleepTimeMilli,
			Percentage: deepPct,
			Color:      theme.ColorSleepDeep,
		},
		{
			Name:       "REM",
			DurationMs: stages.TotalREMSleepTimeMilli,
			Percentage: remPct,
			Color:      theme.ColorSleepREM,
		},
	}

	// calculate baseline sleep duration from historical data
	baselineSleepMs := calculateSleepDurationBaseline(state)

	stagesChart := chart.NewSleepStagesChart(stageData, totalDuration,
		chart.WithSleepStagesBaseline(baselineSleepMs),
	)
	return stagesChart.Render(width)
}

func sleepPerformanceChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(data) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepPerformancePercentage,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := sleepChartTitle("WEEKLY SLEEP PERFORMANCE")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.SleepColor),
		chart.WithBarChartFormatter(chart.FormatPercentage),
		chart.WithBarChartMax(100),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func hoursVsNeededDualChart(state State, width int) string {
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
	if len(actualData) == 0 {
		return ""
	}

	title := sleepChartTitle("HOURS OF SLEEP VS NEEDED")
	c := chart.NewDualLineChart(actualData, neededData,
		chart.WithDualLineColors(theme.ColorSleep, theme.ColorTeal),
		chart.WithDualLineLabels("HOURS OF SLEEP", "SLEEP NEEDED"),
		chart.WithDualLineHeight(5),
		chart.WithDualLineAutoScale(true),
		chart.WithDualLineShowValues(true),
		chart.WithDualLineShowLegend(true),
		chart.WithDualLineLegendPosition(chart.LegendTopRight),
		chart.WithDualLineFormatter(chart.FormatDurationFromHours),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func hoursVsNeededPercentChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(data) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepPerformancePercentage,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := sleepChartTitle("HOURS VS NEEDED (%)")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.SleepColor),
		chart.WithBarChartFormatter(chart.FormatPercentage),
		chart.WithBarChartMax(100),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func restorativeSleepChart(state State, width int) string {
	var data []chart.StackedDataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(data) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		stages := s.Score.StageSummary
		remHours := float64(stages.TotalREMSleepTimeMilli) / (1000 * 60 * 60)
		deepHours := float64(stages.TotalSlowWaveSleepTimeMilli) / (1000 * 60 * 60)
		data = append(data, chart.StackedDataPoint{
			Label:  s.CreatedAt.Format("Mon\n2"),
			Values: []float64{remHours, deepHours},
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := sleepChartTitle("RESTORATIVE SLEEP")
	c := chart.NewStackedBarChart(data,
		chart.WithStackedBarColors([]colorpkg.Color{theme.ColorSleepDeep, theme.ColorSleepREM}),
		chart.WithStackedBarLabels([]string{"DEEP SLEEP", "REM SLEEP"}),
		chart.WithStackedBarFormatter(chart.FormatDurationFromHours),
		chart.WithStackedBarHeight(6),
		chart.WithStackedBarShowLegend(true),
		chart.WithStackedBarLegendPosition(chart.LegendTopRight),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func sleepConsistencyChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(data) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepConsistencyPercentage,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := sleepChartTitle("SLEEP CONSISTENCY (%)")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.SleepColor),
		chart.WithBarChartFormatter(chart.FormatPercentage),
		chart.WithBarChartMax(100),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func sleepEfficiencyChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(data) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.SleepEfficiencyPercentage,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := sleepChartTitle("SLEEP EFFICIENCY (%)")
	c := chart.NewLineChart(data,
		chart.WithLineChartColor(theme.ColorSleep),
		chart.WithLineChartFormatter(chart.FormatPercentage),
		chart.WithLineChartHeight(5),
		chart.WithLineChartShowValues(true),
		chart.WithLineChartShowDots(true),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func sleepDebtChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(data) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		debtHours := float64(s.Score.SleepNeeded.NeedFromSleepDebtMilli) / (1000 * 60 * 60)
		data = append(data, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: debtHours,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := sleepChartTitle("SLEEP DEBT")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.SleepColor),
		chart.WithBarChartFormatter(chart.FormatDurationFromHours),
		chart.WithBarChartMax(3),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func sleepChartTitle(text string) string {
	return lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true).
		Render(text)
}

func renderSleepLegend() string {
	poorBox := lipgloss.NewStyle().Foreground(theme.ColorOrange).Render(theme.SymbolSquareFilled)
	poorText := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(" Poor  ")
	suffBox := lipgloss.NewStyle().Foreground(theme.ColorNeutral).Render(theme.SymbolSquareFilled)
	suffText := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(" Sufficient  ")
	optBox := lipgloss.NewStyle().Foreground(theme.ColorTeal).Render(theme.SymbolSquareFilled)
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
			directionStr = " " + lipgloss.NewStyle().Foreground(theme.ColorTeal).Render(theme.SymbolArrowUp)
		} else if currentRestorativeMs < baselineRestorativeMs {
			directionStr = " " + lipgloss.NewStyle().Foreground(theme.ColorOrange).Render(theme.SymbolArrowDown)
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
		Foreground(theme.ColorSleepREM).
		Background(theme.ColorSleepDeep)
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
