package dashboard

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/tui/components/chart"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/components/lazy_list"
	"github.com/garrettladley/thoop/internal/tui/components/metric_row"
	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/units"
	"github.com/garrettladley/thoop/internal/xtime"
)

func RenderStrainDetail(state *State, width, height int) string {
	metricsWidth := 56
	viewportHeight := height - 2 // reserve space for footer

	items := buildStrainItems(state, metricsWidth)

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

func buildStrainItems(state *State, metricsWidth int) []lazy_list.Item {
	items := make([]lazy_list.Item, 0, 15)

	strainGauge := gauge.New(
		state.StrainScore,
		21,
		"STRAIN",
		theme.ColorStrain,
	)
	items = append(items, lazy_list.NewStaticItem(strainGauge.Render()))

	items = append(items, lazy_list.NewSpacerItem(2))

	metrics := renderStrainMetrics(*state, metricsWidth)
	items = append(items, lazy_list.NewStaticItem(metrics))

	if len(state.TodaysWorkouts) > 0 {
		items = append(items, lazy_list.NewSpacerItem(3))

		activitiesSection := renderActivities(state.TodaysWorkouts, state.SelectedDate, metricsWidth)
		items = append(items, lazy_list.NewStaticItem(activitiesSection))
	}

	if len(state.HistoricalCycles) > 0 {
		items = append(items, lazy_list.NewSpacerItem(3))

		titleStyle := lipgloss.NewStyle().
			Foreground(theme.ColorWhite).
			Bold(true)
		weeklyTrendsTitle := titleStyle.Width(metricsWidth).Render("WEEKLY TRENDS")
		items = append(items, lazy_list.NewStaticItem(weeklyTrendsTitle))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("strain-trend", func(w int) string {
			return strainTrendChart(*state, w)
		}))

		items = append(items, lazy_list.NewSpacerItem(3))

		items = append(items, lazy_list.NewChartItem("strain-calories", func(w int) string {
			return caloriesTrendChart(*state, w)
		}))
	}

	return items
}

func renderStrainMetrics(state State, width int) string {
	var rows []string

	if state.CurrentCycle != nil && state.CurrentCycle.Score != nil {
		score := state.CurrentCycle.Score

		strainAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.Strain })
		strainDirection := getDirectionHigherBetter(score.Strain, strainAvg, WithPrecision(1))
		strainRow := metric_row.New(
			"Strain",
			fmt.Sprintf("%.1f", score.Strain),
			width,
			metric_row.WithDirection(strainDirection),
			metric_row.WithSubValue(formatAvg(strainAvg, "%.1f")),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, strainRow.Render())
		rows = append(rows, "")

		calories := units.KilojoulesToCalories(score.Kilojoule)
		caloriesAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.Calories })
		caloriesDirection := getDirectionHigherBetter(calories, caloriesAvg, WithPrecision(0))
		caloriesRow := metric_row.New(
			"Calories",
			units.FormatWithCommas(calories),
			width,
			metric_row.WithDirection(caloriesDirection),
			metric_row.WithSubValue(formatAvgWithCommas(caloriesAvg)),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, caloriesRow.Render())
		rows = append(rows, "")

		avgHR := float64(score.AverageHeartRate)
		avgHRAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.AvgHeartRate })
		avgHRDirection := getDirectionHigherBetter(avgHR, avgHRAvg, WithPrecision(0))
		avgHRRow := metric_row.New(
			"Avg Heart Rate",
			fmt.Sprintf("%.0f", avgHR),
			width,
			metric_row.WithDirection(avgHRDirection),
			metric_row.WithSubValue(formatAvg(avgHRAvg, "%.0f")),
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, avgHRRow.Render())
		rows = append(rows, "")

		maxHR := float64(score.MaxHeartRate)
		maxHRAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.MaxHeartRate })
		maxHRDirection := getDirectionHigherBetter(maxHR, maxHRAvg, WithPrecision(0))
		maxHRRow := metric_row.New(
			"Max Heart Rate",
			fmt.Sprintf("%.0f", maxHR),
			width,
			metric_row.WithDirection(maxHRDirection),
			metric_row.WithSubValue(formatAvg(maxHRAvg, "%.0f")),
			metric_row.WithLabelColor(theme.ColorWhite),
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

func strainTrendChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalCycles) - 1; i >= 0 && len(data) < 7; i-- {
		c := state.HistoricalCycles[i]
		if c.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: c.Start.Format("Mon\n2"),
			Value: c.Score.Strain,
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := strainChartTitle("STRAIN")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.StrainColor),
		chart.WithBarChartFormatter(chart.FormatFloat1),
		chart.WithBarChartMax(21),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func caloriesTrendChart(state State, width int) string {
	var data []chart.DataPoint
	for i := len(state.HistoricalCycles) - 1; i >= 0 && len(data) < 7; i-- {
		c := state.HistoricalCycles[i]
		if c.Score == nil {
			continue
		}
		data = append(data, chart.DataPoint{
			Label: c.Start.Format("Mon\n2"),
			Value: units.KilojoulesToCalories(c.Score.Kilojoule),
		})
	}
	if len(data) == 0 {
		return ""
	}

	title := strainChartTitle("CALORIES")
	c := chart.NewBarChart(data,
		chart.WithBarChartColorFunc(chart.StaticColor(theme.ColorStrain)),
		chart.WithBarChartFormatter(chart.FormatIntWithCommas),
		chart.WithBarChartHeight(6),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, "", c.Render(width))
}

func strainChartTitle(text string) string {
	return lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true).
		Render(text)
}

func activitiesTitle(selectedDate *time.Time) string {
	if selectedDate == nil || xtime.IsToday(*selectedDate) {
		return "TODAY'S ACTIVITIES"
	}
	return selectedDate.Format("MON JAN 2") + " ACTIVITIES"
}

func renderActivities(workouts []whoop.Workout, selectedDate *time.Time, width int) string {
	if len(workouts) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true)

	sections := make([]string, 0, len(workouts)+2)
	sections = append(sections, titleStyle.Render(activitiesTitle(selectedDate)))
	sections = append(sections, "")

	for _, w := range workouts {
		activityRow := renderActivityRow(w, width)
		sections = append(sections, activityRow)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderActivityRow(w whoop.Workout, width int) string {
	var strainStr string
	if w.Score != nil {
		strainStr = fmt.Sprintf("%4.1f", w.Score.Strain)
	} else {
		strainStr = "--"
	}

	badgeStyle := lipgloss.NewStyle().
		Background(theme.ColorStrain).
		Foreground(theme.ColorWhite).
		Bold(true).
		Align(lipgloss.Center).
		Width(6)

	badge := badgeStyle.Render(strainStr)

	nameStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true).
		MarginLeft(2)

	name := nameStyle.Render(strings.ToUpper(w.SportName))

	timeRange := formatTimeRange(w.Start, w.End)
	timeStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	leftContent := lipgloss.JoinHorizontal(lipgloss.Center, badge, name)
	leftWidth := lipgloss.Width(leftContent)
	timeWidth := lipgloss.Width(timeRange)
	spacing := max(width-leftWidth-timeWidth, 2)

	spacer := strings.Repeat(" ", spacing)

	return leftContent + spacer + timeStyle.Render(timeRange)
}

func formatTimeRange(start, end time.Time) string {
	const timeFormat = "3:04 PM"
	startStr := fmt.Sprintf("%8s", start.Local().Format(timeFormat))
	endStr := fmt.Sprintf("%8s", end.Local().Format(timeFormat))
	return fmt.Sprintf("%s - %s", startStr, endStr)
}

func formatAvgWithCommas(avg float64) string {
	if avg == 0 {
		return ""
	}
	return units.FormatWithCommas(avg)
}
