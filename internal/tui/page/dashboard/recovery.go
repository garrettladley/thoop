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

func RenderRecoveryDetail(state State, width, height int) string {
	recoveryGauge := gauge.New(
		state.RecoveryScore,
		100,
		"RECOVERY",
		recoveryColor(state.RecoveryScore),
	)

	gaugeStr := recoveryGauge.Render()

	metricsWidth := 40
	metrics := renderRecoveryMetrics(state, metricsWidth)

	var contentParts []string
	contentParts = append(contentParts, gaugeStr)
	contentParts = append(contentParts, "", "")
	contentParts = append(contentParts, metrics)

	chartsSection := renderRecoveryCharts(state, metricsWidth)
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

	// apply viewport scrolling
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

func renderRecoveryMetrics(state State, width int) string {
	var rows []string

	if state.CurrentRecovery != nil && state.CurrentRecovery.Score != nil {
		score := state.CurrentRecovery.Score

		// HRV: Higher is better
		hrvAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.HRV })
		hrvDirection := getDirectionHigherBetter(score.HRVRmssdMilli, hrvAvg)
		hrvRow := metric_row.New(
			"HRV",
			fmt.Sprintf("%.0f", score.HRVRmssdMilli),
			width,
			metric_row.WithDirection(hrvDirection),
			metric_row.WithSubValue(formatAvg(hrvAvg, "%.0f")),
		)
		rows = append(rows, hrvRow.Render())
		rows = append(rows, "")

		rhrAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.RestingHeartRate })
		rhrDirection := getDirectionLowerBetter(score.RestingHeartRate, rhrAvg)
		rhrRow := metric_row.New(
			"Resting Heart Rate",
			fmt.Sprintf("%.0f", score.RestingHeartRate),
			width,
			metric_row.WithDirection(rhrDirection),
			metric_row.WithSubValue(formatAvg(rhrAvg, "%.0f")),
		)
		rows = append(rows, rhrRow.Render())
		rows = append(rows, "")

		if state.CurrentSleep != nil && state.CurrentSleep.Score != nil {
			respRate := state.CurrentSleep.Score.RespiratoryRate
			respAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.RespiratoryRate })
			respDirection := getDirectionLowerBetter(respRate, respAvg)
			respRow := metric_row.New(
				"Respiratory Rate",
				fmt.Sprintf("%.1f", respRate),
				width,
				metric_row.WithDirection(respDirection),
				metric_row.WithSubValue(formatAvg(respAvg, "%.1f")),
			)
			rows = append(rows, respRow.Render())
			rows = append(rows, "")

			sleepPerf := state.CurrentSleep.Score.SleepPerformancePercentage
			sleepAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.SleepPerformance })
			sleepDirection := getDirectionHigherBetter(sleepPerf, sleepAvg)
			sleepRow := metric_row.New(
				"Sleep Performance",
				fmt.Sprintf("%.0f", sleepPerf),
				width,
				metric_row.WithDirection(sleepDirection),
				metric_row.WithSubValue(formatAvg(sleepAvg, "%.0f")),
				metric_row.WithUnit("%"),
			)
			rows = append(rows, sleepRow.Render())
			rows = append(rows, "")
		}

		rows = append(rows, "")
		rows = append(rows, renderComparisonLegend())
	} else {
		noDataStyle := lipgloss.NewStyle().
			Foreground(theme.ColorDim).
			Italic(true)
		rows = append(rows, noDataStyle.Render("No recovery data available"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderRecoveryCharts(state State, width int) string {
	if len(state.HistoricalRecoveries) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true)

	var sections []string
	sections = append(sections, titleStyle.Render("WEEKLY RECOVERY"))
	sections = append(sections, "")

	var recoveryData []chart.DataPoint
	for i := len(state.HistoricalRecoveries) - 1; i >= 0 && len(recoveryData) < 7; i-- {
		r := state.HistoricalRecoveries[i]
		if r.Score == nil {
			continue
		}
		recoveryData = append(recoveryData, chart.DataPoint{
			Label: r.CreatedAt.Format("Mon\n2"),
			Value: r.Score.RecoveryScore,
		})
	}

	for i, j := 0, len(recoveryData)-1; i < j; i, j = i+1, j-1 {
		recoveryData[i], recoveryData[j] = recoveryData[j], recoveryData[i]
	}

	if len(recoveryData) > 0 {
		recoveryChart := chart.NewBarChart(recoveryData,
			chart.WithBarChartColorFunc(chart.RecoveryColor),
			chart.WithBarChartFormatter(chart.FormatPercentage),
			chart.WithBarChartMax(100),
			chart.WithBarChartHeight(6),
			chart.WithBarChartShowValues(true),
		)
		sections = append(sections, recoveryChart.Render(width))
	}

	// HRV trend line chart
	sections = append(sections, "")
	sections = append(sections, "")
	sections = append(sections, titleStyle.Render("HRV TREND"))
	sections = append(sections, "")

	var hrvData []chart.DataPoint
	for i := len(state.HistoricalRecoveries) - 1; i >= 0 && len(hrvData) < 7; i-- {
		r := state.HistoricalRecoveries[i]
		if r.Score == nil {
			continue
		}
		hrvData = append(hrvData, chart.DataPoint{
			Label: r.CreatedAt.Format("Mon\n2"),
			Value: r.Score.HRVRmssdMilli,
		})
	}

	// reverse for chronological order
	for i, j := 0, len(hrvData)-1; i < j; i, j = i+1, j-1 {
		hrvData[i], hrvData[j] = hrvData[j], hrvData[i]
	}

	if len(hrvData) > 0 {
		hrvChart := chart.NewLineChart(hrvData,
			chart.WithLineChartColor(theme.ColorRecoveryBlue),
			chart.WithLineChartFormatter(chart.FormatInt),
			chart.WithLineChartHeight(5),
			chart.WithLineChartShowValues(true),
			chart.WithLineChartShowDots(true),
		)
		sections = append(sections, hrvChart.Render(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderComparisonLegend() string {
	upArrow := lipgloss.NewStyle().Foreground(theme.ColorTeal).Render("▲")
	downArrow := lipgloss.NewStyle().Foreground(theme.ColorOrange).Render("▼")
	text := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(" Today vs. last 30 days")
	return upArrow + downArrow + text
}

func getAvgValue(averages *ThirtyDayAverages, getAvg func(*ThirtyDayAverages) float64) float64 {
	if averages == nil {
		return 0
	}
	return getAvg(averages)
}

func formatAvg(avg float64, format string) string {
	if avg == 0 {
		return ""
	}
	return fmt.Sprintf(format, avg)
}

// getDirectionHigherBetter returns direction for metrics where higher is better (HRV, Sleep Performance)
func getDirectionHigherBetter(current, avg float64) metric_row.Direction {
	if avg == 0 {
		return metric_row.DirectionNone
	}

	if current > avg {
		return metric_row.DirectionUp // teal up - higher is good
	} else if current < avg {
		return metric_row.DirectionDown // orange down - lower is bad
	}
	return metric_row.DirectionNeutral
}

// getDirectionLowerBetter returns direction for metrics where lower is better (RHR, Respiratory Rate)
func getDirectionLowerBetter(current, avg float64) metric_row.Direction {
	if avg == 0 {
		return metric_row.DirectionNone
	}

	if current > avg {
		return metric_row.DirectionUpBad // orange up - higher is bad
	} else if current < avg {
		return metric_row.DirectionDownGood // teal down - lower is good
	}
	return metric_row.DirectionNeutral
}
