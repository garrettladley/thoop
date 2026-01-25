package dashboard

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/tui/components/chart"
	"github.com/garrettladley/thoop/internal/tui/components/gauge"
	"github.com/garrettladley/thoop/internal/tui/components/metric_row"
	"github.com/garrettladley/thoop/internal/tui/components/viewport"
	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/units"
	"github.com/garrettladley/thoop/internal/xtime"
)

func RenderStrainDetail(state State, width, height int) string {
	strainGauge := gauge.New(
		state.StrainScore,
		21,
		"STRAIN",
		theme.ColorStrain,
	)

	gaugeStr := strainGauge.Render()

	metricsWidth := 56
	metrics := renderStrainMetrics(state, metricsWidth)

	var contentParts []string
	contentParts = append(contentParts, gaugeStr)
	contentParts = append(contentParts, "", "")
	contentParts = append(contentParts, metrics)

	activitiesSection := renderActivities(state.TodaysWorkouts, state.SelectedDate, metricsWidth)
	if activitiesSection != "" {
		contentParts = append(contentParts, "", "", "")
		contentParts = append(contentParts, activitiesSection)
	}

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
			metric_row.WithLabelColor(theme.ColorWhite),
		)
		rows = append(rows, strainRow.Render())
		rows = append(rows, "")

		calories := units.KilojoulesToCalories(score.Kilojoule)
		caloriesAvg := getAvgValue(state.Averages, func(a *ThirtyDayAverages) float64 { return a.Calories })
		caloriesDirection := getDirectionHigherBetter(calories, caloriesAvg)
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
		avgHRDirection := getDirectionHigherBetter(avgHR, avgHRAvg)
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
		maxHRDirection := getDirectionHigherBetter(maxHR, maxHRAvg)
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

func renderStrainCharts(state State, width int) string {
	if len(state.HistoricalCycles) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true)
	weeklyTrendsTitle := titleStyle.Width(width).Render("WEEKLY TRENDS")

	var sections []string
	sections = append(sections, weeklyTrendsTitle)

	if c := strainTrendChart(state, width); c != "" {
		sections = append(sections, "", "", c)
	}
	if c := caloriesTrendChart(state, width); c != "" {
		sections = append(sections, "", "", c)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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
	return selectedDate.Format("Mon Jan 2") + " ACTIVITIES"
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
		strainStr = fmt.Sprintf("%.1f", w.Score.Strain)
	} else {
		strainStr = "--"
	}

	badgeStyle := lipgloss.NewStyle().
		Background(theme.ColorStrain).
		Foreground(theme.ColorWhite).
		Bold(true).
		Padding(0, 1)

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
	return fmt.Sprintf("%s - %s", start.Local().Format(timeFormat), end.Local().Format(timeFormat))
}

func formatAvgWithCommas(avg float64) string {
	if avg == 0 {
		return ""
	}
	return units.FormatWithCommas(avg)
}
