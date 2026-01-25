package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// StackedBarChart renders a stacked vertical bar chart.
type StackedBarChart struct {
	data         []StackedDataPoint
	height       int
	colors       []color.Color
	labels       []string // legend labels for each segment
	formatter    ValueFormatter
	showValues   bool
	showAxis     bool
	showLegend   bool
	barWidth     int
	barGap       int
	bgColor      color.Color
	showSegTotal bool // show total value above bars
}

// StackedBarChartOption configures a StackedBarChart.
type StackedBarChartOption func(*StackedBarChart)

// WithStackedBarHeight sets the bar height.
func WithStackedBarHeight(h int) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.height = h
	}
}

// WithStackedBarColors sets the segment colors.
func WithStackedBarColors(colors []color.Color) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.colors = colors
	}
}

// WithStackedBarLabels sets the legend labels.
func WithStackedBarLabels(labels []string) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.labels = labels
	}
}

// WithStackedBarFormatter sets the value formatter.
func WithStackedBarFormatter(f ValueFormatter) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.formatter = f
	}
}

// WithStackedBarShowValues shows values above bars.
func WithStackedBarShowValues(show bool) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.showValues = show
	}
}

// WithStackedBarShowAxis shows the x-axis labels.
func WithStackedBarShowAxis(show bool) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.showAxis = show
	}
}

// WithStackedBarShowLegend shows the legend.
func WithStackedBarShowLegend(show bool) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.showLegend = show
	}
}

// WithStackedBarBarWidth sets the bar width.
func WithStackedBarBarWidth(w int) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.barWidth = w
	}
}

// WithStackedBarBarGap sets the gap between bars.
func WithStackedBarBarGap(g int) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.barGap = g
	}
}

// WithStackedBarBgColor sets the background color.
func WithStackedBarBgColor(c color.Color) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.bgColor = c
	}
}

// WithStackedBarShowTotal shows total value above bars.
func WithStackedBarShowTotal(show bool) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.showSegTotal = show
	}
}

// NewStackedBarChart creates a new stacked bar chart.
func NewStackedBarChart(data []StackedDataPoint, opts ...StackedBarChartOption) *StackedBarChart {
	sbc := &StackedBarChart{
		data:       data,
		height:     8,
		formatter:  FormatInt,
		showValues: true,
		showAxis:   true,
		showLegend: true,
		barWidth:   3,
		barGap:     2,
		bgColor:    theme.ColorBgLight,
	}
	for _, opt := range opts {
		opt(sbc)
	}
	return sbc
}

// Render renders the stacked bar chart.
func (sbc *StackedBarChart) Render(width int) string {
	if len(sbc.data) == 0 || width <= 0 {
		return ""
	}

	// find max total for scaling
	maxTotal := 0.0
	for _, d := range sbc.data {
		total := 0.0
		for _, v := range d.Values {
			total += v
		}
		maxTotal = max(maxTotal, total)
	}
	if maxTotal <= 0 {
		maxTotal = 100
	}

	// calculate bar dimensions
	numBars := len(sbc.data)
	totalBarSpace := numBars*sbc.barWidth + (numBars-1)*sbc.barGap

	barWidth := sbc.barWidth
	barGap := sbc.barGap
	if totalBarSpace > width {
		availPerBar := width / numBars
		barWidth = max(1, availPerBar-1)
		barGap = max(0, availPerBar-barWidth)
		totalBarSpace = numBars*barWidth + (numBars-1)*barGap
	}

	leftPad := (width - totalBarSpace) / 2

	var sections []string
	// value labels above bars
	if sbc.showValues {
		valueRow := strings.Repeat(" ", leftPad)
		for i, d := range sbc.data {
			total := 0.0
			for _, v := range d.Values {
				total += v
			}
			valueStr := sbc.formatter(total)
			padded := centerString(valueStr, barWidth)
			valueRow += padded
			if i < numBars-1 {
				valueRow += strings.Repeat(" ", barGap)
			}
		}
		valueStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)
		sections = append(sections, valueStyle.Render(valueRow))
	}

	// render stacked bars
	barStrings := make([][]string, numBars)
	for i, d := range sbc.data {
		// calculate fill percentages for each segment
		total := 0.0
		for _, v := range d.Values {
			total += v
		}

		var segments []Segment
		for j, v := range d.Values {
			fillPct := v / maxTotal
			if fillPct > 0 {
				c := sbc.bgColor
				if j < len(sbc.colors) {
					c = sbc.colors[j]
				}
				segments = append(segments, Segment{Value: fillPct, Color: c})
			}
		}

		bar := NewBar(segments,
			WithBarHeight(sbc.height),
			WithBarWidth(barWidth),
			WithBarBgColor(sbc.bgColor),
		)
		barStrings[i] = strings.Split(bar.Render(), "\n")
	}

	// combine bars horizontally
	for row := range sbc.height {
		rowStr := strings.Repeat(" ", leftPad)
		for i := range numBars {
			if row < len(barStrings[i]) {
				rowStr += barStrings[i][row]
			} else {
				rowStr += strings.Repeat(" ", barWidth)
			}
			if i < numBars-1 {
				rowStr += strings.Repeat(" ", barGap)
			}
		}
		sections = append(sections, rowStr)
	}

	// X-axis labels
	if sbc.showAxis {
		labels := make([]string, numBars)
		for i, d := range sbc.data {
			labels[i] = d.Label
		}
		axis := NewAxis(labels, totalBarSpace, WithAxisTextColor(theme.ColorDim))
		axisStr := axis.Render()
		axisLines := strings.Split(axisStr, "\n")
		for _, line := range axisLines {
			sections = append(sections, strings.Repeat(" ", leftPad)+line)
		}
	}

	if sbc.showLegend && len(sbc.labels) > 0 {
		var items []LegendItem
		for i, label := range sbc.labels {
			c := sbc.bgColor
			if i < len(sbc.colors) {
				c = sbc.colors[i]
			}
			items = append(items, LegendItem{Label: label, Color: c})
		}
		legend := NewLegend(items)
		sections = append(sections, "")
		sections = append(sections, legend.Render())
	}

	return strings.Join(sections, "\n")
}
