package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// LineChart renders a line chart with axis labels and optional value display.
type LineChart struct {
	data       []DataPoint
	height     int
	minValue   float64
	maxValue   float64
	color      color.Color
	formatter  ValueFormatter
	showValues bool
	showAxis   bool
	showDots   bool
}

// LineChartOption configures a LineChart.
type LineChartOption func(*LineChart)

// WithLineChartHeight sets the chart height.
func WithLineChartHeight(h int) LineChartOption {
	return func(lc *LineChart) {
		lc.height = h
	}
}

// WithLineChartMin sets the minimum value for scaling.
func WithLineChartMin(min float64) LineChartOption {
	return func(lc *LineChart) {
		lc.minValue = min
	}
}

// WithLineChartMax sets the maximum value for scaling.
func WithLineChartMax(max float64) LineChartOption {
	return func(lc *LineChart) {
		lc.maxValue = max
	}
}

// WithLineChartColor sets the line color.
func WithLineChartColor(c color.Color) LineChartOption {
	return func(lc *LineChart) {
		lc.color = c
	}
}

// WithLineChartFormatter sets the value formatter.
func WithLineChartFormatter(f ValueFormatter) LineChartOption {
	return func(lc *LineChart) {
		lc.formatter = f
	}
}

// WithLineChartShowValues shows values at data points.
func WithLineChartShowValues(show bool) LineChartOption {
	return func(lc *LineChart) {
		lc.showValues = show
	}
}

// WithLineChartShowAxis shows the x-axis labels.
func WithLineChartShowAxis(show bool) LineChartOption {
	return func(lc *LineChart) {
		lc.showAxis = show
	}
}

// WithLineChartShowDots shows dots at data points.
func WithLineChartShowDots(show bool) LineChartOption {
	return func(lc *LineChart) {
		lc.showDots = show
	}
}

// NewLineChart creates a new line chart.
func NewLineChart(data []DataPoint, opts ...LineChartOption) *LineChart {
	lc := &LineChart{
		data:       data,
		height:     6,
		minValue:   0,
		maxValue:   0,
		color:      theme.ColorTeal,
		formatter:  FormatInt,
		showValues: true,
		showAxis:   true,
		showDots:   true,
	}
	for _, opt := range opts {
		opt(lc)
	}

	// Auto-detect min/max if not set
	if lc.maxValue <= lc.minValue {
		for _, d := range lc.data {
			if d.Value > lc.maxValue {
				lc.maxValue = d.Value
			}
			if d.Value < lc.minValue || lc.minValue == 0 {
				lc.minValue = d.Value
			}
		}
		// Add some padding
		if lc.minValue == lc.maxValue {
			lc.minValue = 0
			if lc.maxValue == 0 {
				lc.maxValue = 100
			}
		}
	}

	return lc
}

// Render renders the line chart.
func (lc *LineChart) Render(width int) string {
	if len(lc.data) == 0 || width <= 0 {
		return ""
	}

	var sections []string

	// value labels above data points
	if lc.showValues {
		valueRow := renderValueLabels(lc.data, width, lc.formatter)
		sections = append(sections, valueRow)
	}

	// extract values and normalize
	values := make([]float64, len(lc.data))
	for i, d := range lc.data {
		values[i] = d.Value
	}
	normalized := NormalizeValues(values, lc.minValue, lc.maxValue)

	// render line
	line := NewLine(normalized, width,
		WithLineColor(lc.color),
		WithLineHeight(lc.height),
		WithLineShowDots(lc.showDots),
	)
	sections = append(sections, line.Render())

	// X-axis labels
	if lc.showAxis {
		labels := make([]string, len(lc.data))
		for i, d := range lc.data {
			labels[i] = d.Label
		}
		axis := NewAxis(labels, width, WithAxisTextColor(theme.ColorDim))
		sections = append(sections, axis.Render())
	}

	return strings.Join(sections, "\n")
}

// renderValueLabels renders value labels above data points.
func renderValueLabels(data []DataPoint, width int, formatter ValueFormatter) string {
	if len(data) == 0 {
		return ""
	}

	// calculate positions for each value
	step := float64(width) / float64(len(data))
	row := make([]rune, width)
	for i := range row {
		row[i] = ' '
	}

	for i, d := range data {
		valueStr := formatter(d.Value)
		pos := int(float64(i)*step + step/2)
		startPos := pos - len(valueStr)/2

		if startPos < 0 {
			startPos = 0
		}
		if startPos+len(valueStr) > width {
			startPos = width - len(valueStr)
		}

		for j, r := range valueStr {
			if startPos+j < width {
				row[startPos+j] = r
			}
		}
	}

	style := lipgloss.NewStyle().Foreground(theme.ColorWhite)
	return style.Render(string(row))
}
