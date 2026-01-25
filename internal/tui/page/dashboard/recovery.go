package dashboard

import (
	"fmt"
	"math"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/chart"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/components/lazy_list"
	"github.com/garrettladley/thoop/internal/tui/components/metric_row"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

func RenderRecoveryDetail(state *State, width, height int) string {
	metricsWidth := 56
	viewportHeight := height - 2 // reserve space for footer

	items := buildRecoveryItems(state, metricsWidth)

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

	// apply viewport scrolling and sync state
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

func buildRecoveryItems(state *State, metricsWidth int) []lazy_list.Item {
	items := make([]lazy_list.Item, 0, 15)

	recoveryGauge := gauge.New(
		state.RecoveryScore,
		100,
		"RECOVERY",
		recoveryColor(state.RecoveryScore),
	)
	items = append(items, lazy_list.NewStaticItem(recoveryGauge.Render()))

	items = append(items, lazy_list.NewSpacerItem(2))

	metrics := renderRecoveryMetrics(*state, metricsWidth)
	items = append(items, lazy_list.NewStaticItem(metrics))

	if len(state.HistoricalRecoveries) > 0 {
		items = append(items, lazy_list.NewSpacerItem(3))

		titleStyle := lipgloss.NewStyle().
			Foreground(theme.ColorWhite).
			Bold(true)
		weeklyTrendsTitle := titleStyle.Width(metricsWidth).Render("WEEKLY TRENDS")
		items = append(items, lazy_list.NewStaticItem(weeklyTrendsTitle))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("recovery-trend", func(w int) string {
			return recoveryTrendChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("recovery-hrv", func(w int) string {
			return hrvTrendChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("recovery-rhr", func(w int) string {
			return restingHeartRateChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("recovery-resp", func(w int) string {
			return respiratoryRateChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("recovery-sleep-perf", func(w int) string {
			return recoverySleepPerfChart(*state, w)
		}))
	}

	return items
}

func renderRecoveryMetrics(state State, width int) string {
	var rows []string

	if state.CurrentRecovery != nil && state.CurrentRecovery.Score != nil {
		score := state.CurrentRecovery.Score

		hrvAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.HRV })
		hrvDirection := getDirectionHigherBetter(score.HRVRmssdMilli, hrvAvg, WithPrecision(0))
		hrvRow := metric_row.New(
			"HRV",
			fmt.Sprintf("%.0f", score.HRVRmssdMilli),
			width,
			metric_row.WithDirection(hrvDirection),
			metric_row.WithSubValue(formatAvg(hrvAvg, "%.0f")),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, hrvRow.Render())
		rows = append(rows, "")

		rhrAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.RestingHeartRate })
		rhrDirection := getDirectionLowerBetter(score.RestingHeartRate, rhrAvg, WithPrecision(0))
		rhrRow := metric_row.New(
			"Resting Heart Rate",
			fmt.Sprintf("%.0f", score.RestingHeartRate),
			width,
			metric_row.WithDirection(rhrDirection),
			metric_row.WithSubValue(formatAvg(rhrAvg, "%.0f")),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, rhrRow.Render())
		rows = append(rows, "")

		if state.CurrentSleep != nil && state.CurrentSleep.Score != nil {
			respRate := state.CurrentSleep.Score.RespiratoryRate
			respAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.RespiratoryRate })
			respDirection := getDirectionLowerBetter(respRate, respAvg, WithPrecision(1))
			respRow := metric_row.New(
				"Respiratory Rate",
				fmt.Sprintf("%.1f", respRate),
				width,
				metric_row.WithDirection(respDirection),
				metric_row.WithSubValue(formatAvg(respAvg, "%.1f")),
				metric_row.WithLabelColor(theme.ColorWhite),
			)
			rows = append(rows, respRow.Render())
			rows = append(rows, "")

			sleepPerf := state.CurrentSleep.Score.SleepPerformancePercentage
			sleepAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.SleepPerformance })
			sleepDirection := getDirectionHigherBetter(sleepPerf, sleepAvg, WithPrecision(0))
			sleepRow := metric_row.New(
				"Sleep Performance",
				fmt.Sprintf("%.0f", math.Round(sleepPerf)),
				width,
				metric_row.WithDirection(sleepDirection),
				metric_row.WithSubValue(formatAvg(sleepAvg, "%.0f")),
				metric_row.WithUnit("%"),
				metric_row.WithLabelColor(theme.ColorWhite),
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

func recoveryTrendChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalRecoveries) - 1; i >= 0 && len(data) < 7; i-- {
		r := state.HistoricalRecoveries[i]
		if r.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: r.CreatedAt.Format("Mon\n2"),
			Value: r.Score.RecoveryScore,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := recoveryChartTitle("RECOVERY")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.RecoveryColor),
		chart.WithBarChartFormatter(chart.FormatPercentage),
		chart.WithBarChartMax(100),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func hrvTrendChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalRecoveries) - 1; i >= 0 && len(data) < 7; i-- {
		r := state.HistoricalRecoveries[i]
		if r.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: r.CreatedAt.Format("Mon\n2"),
			Value: r.Score.HRVRmssdMilli,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := recoveryChartTitle("HRV TREND")
	c := chart.NewLineChart(data,
		chart.WithLineChartColor(theme.ColorRecoveryBlue),
		chart.WithLineChartFormatter(chart.FormatInt),
		chart.WithLineChartHeight(5),
		chart.WithLineChartShowValues(true),
		chart.WithLineChartShowDots(true),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func restingHeartRateChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalRecoveries) - 1; i >= 0 && len(data) < 7; i-- {
		r := state.HistoricalRecoveries[i]
		if r.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: r.CreatedAt.Format("Mon\n2"),
			Value: r.Score.RestingHeartRate,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := recoveryChartTitle("RESTING HEART RATE")
	c := chart.NewLineChart(data,
		chart.WithLineChartColor(theme.ColorRecoveryBlue),
		chart.WithLineChartFormatter(chart.FormatInt),
		chart.WithLineChartHeight(5),
		chart.WithLineChartShowValues(true),
		chart.WithLineChartShowDots(true),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func respiratoryRateChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalSleeps) - 1; i >= 0 && len(data) < 7; i-- {
		s := state.HistoricalSleeps[i]
		if s.Nap || s.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: s.CreatedAt.Format("Mon\n2"),
			Value: s.Score.RespiratoryRate,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := recoveryChartTitle("RESPIRATORY RATE")
	c := chart.NewLineChart(data,
		chart.WithLineChartColor(theme.ColorRecoveryBlue),
		chart.WithLineChartFormatter(chart.FormatFloat1),
		chart.WithLineChartHeight(5),
		chart.WithLineChartShowValues(true),
		chart.WithLineChartShowDots(true),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func recoverySleepPerfChart(state State, width int) string {
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

	title := recoveryChartTitle("SLEEP PERFORMANCE")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.SleepColor),
		chart.WithBarChartFormatter(chart.FormatPercentage),
		chart.WithBarChartMax(100),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func recoveryChartTitle(text string) string {
	return lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true).
		Render(text)
}

func renderComparisonLegend() string {
	upArrow := lipgloss.NewStyle().Foreground(theme.ColorTeal).Render(theme.SymbolArrowUp)
	downArrow := lipgloss.NewStyle().Foreground(theme.ColorOrange).Render(theme.SymbolArrowDown)
	today := lipgloss.NewStyle().Foreground(theme.ColorWhite).Render(" Today")
	rest := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(" vs. last 30 days")
	return upArrow + downArrow + today + rest
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

// DirectionOption configures direction comparison behavior.
type DirectionOption func(*directionConfig)

type directionConfig struct {
	rounder func(float64) float64
}

// WithPrecision rounds values to the specified number of decimal places before comparing.
// This ensures the direction indicator matches what the user sees displayed.
func WithPrecision(decimals int) DirectionOption {
	return func(c *directionConfig) {
		mult := math.Pow(10, float64(decimals))
		c.rounder = func(v float64) float64 {
			return math.Round(v*mult) / mult
		}
	}
}

func applyDirectionOptions(opts []DirectionOption) directionConfig {
	cfg := directionConfig{
		rounder: func(v float64) float64 { return v },
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// getDirectionHigherBetter returns direction for metrics where higher is better (HRV, Sleep Performance)
func getDirectionHigherBetter(current, avg float64, opts ...DirectionOption) metric_row.Direction {
	if avg == 0 {
		return metric_row.DirectionNone
	}

	cfg := applyDirectionOptions(opts)
	roundedCurrent := cfg.rounder(current)
	roundedAvg := cfg.rounder(avg)

	if roundedCurrent > roundedAvg {
		return metric_row.DirectionUp // teal up - higher is good
	} else if roundedCurrent < roundedAvg {
		return metric_row.DirectionDown // orange down - lower is bad
	}
	return metric_row.DirectionNeutral
}

// getDirectionLowerBetter returns direction for metrics where lower is better (RHR, Respiratory Rate)
func getDirectionLowerBetter(current, avg float64, opts ...DirectionOption) metric_row.Direction {
	if avg == 0 {
		return metric_row.DirectionNone
	}

	cfg := applyDirectionOptions(opts)
	roundedCurrent := cfg.rounder(current)
	roundedAvg := cfg.rounder(avg)

	if roundedCurrent > roundedAvg {
		return metric_row.DirectionUpBad // orange up - higher is bad
	} else if roundedCurrent < roundedAvg {
		return metric_row.DirectionDownGood // teal down - lower is good
	}
	return metric_row.DirectionNeutral
}
