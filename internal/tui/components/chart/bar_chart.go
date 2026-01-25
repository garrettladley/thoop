package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// BarChart renders a vertical bar chart with labels and optional values.
type BarChart struct {
	data       []DataPoint
	height     int
	maxValue   float64
	colorFunc  ColorFunc
	formatter  ValueFormatter
	showValues bool
	showAxis   bool
	barWidth   int
	barGap     int
	bgColor    color.Color
}

// BarChartOption configures a BarChart.
type BarChartOption func(*BarChart)

// WithBarChartHeight sets the bar height in rows.
func WithBarChartHeight(h int) BarChartOption {
	return func(bc *BarChart) {
		bc.height = h
	}
}

// WithBarChartMax sets the maximum value for scaling.
func WithBarChartMax(max float64) BarChartOption {
	return func(bc *BarChart) {
		bc.maxValue = max
	}
}

// WithBarChartColorFunc sets the color function.
func WithBarChartColorFunc(f ColorFunc) BarChartOption {
	return func(bc *BarChart) {
		bc.colorFunc = f
	}
}

// WithBarChartFormatter sets the value formatter.
func WithBarChartFormatter(f ValueFormatter) BarChartOption {
	return func(bc *BarChart) {
		bc.formatter = f
	}
}

// WithBarChartShowValues shows values above bars.
func WithBarChartShowValues(show bool) BarChartOption {
	return func(bc *BarChart) {
		bc.showValues = show
	}
}

// WithBarChartShowAxis shows the x-axis labels.
func WithBarChartShowAxis(show bool) BarChartOption {
	return func(bc *BarChart) {
		bc.showAxis = show
	}
}

// WithBarChartBarWidth sets the width of each bar.
func WithBarChartBarWidth(w int) BarChartOption {
	return func(bc *BarChart) {
		bc.barWidth = w
	}
}

// WithBarChartBarGap sets the gap between bars.
func WithBarChartBarGap(g int) BarChartOption {
	return func(bc *BarChart) {
		bc.barGap = g
	}
}

// WithBarChartBgColor sets the background color for unfilled portions.
func WithBarChartBgColor(c color.Color) BarChartOption {
	return func(bc *BarChart) {
		bc.bgColor = c
	}
}

// NewBarChart creates a new bar chart.
func NewBarChart(data []DataPoint, opts ...BarChartOption) *BarChart {
	bc := &BarChart{
		data:       data,
		height:     8,
		maxValue:   0,
		colorFunc:  StaticColor(theme.ColorTeal),
		formatter:  FormatInt,
		showValues: true,
		showAxis:   true,
		barWidth:   3,
		barGap:     2,
		bgColor:    theme.ColorBgLight,
	}
	for _, opt := range opts {
		opt(bc)
	}

	// auto-detect max if not set
	if bc.maxValue <= 0 {
		for _, d := range bc.data {
			if d.Value > bc.maxValue {
				bc.maxValue = d.Value
			}
		}
		if bc.maxValue <= 0 {
			bc.maxValue = 100
		}
	}

	return bc
}

// Render renders the bar chart to fit within the given width.
func (bc *BarChart) Render(width int) string {
	if len(bc.data) == 0 || width <= 0 {
		return ""
	}

	// calculate bar dimensions to fit width
	numBars := len(bc.data)
	totalBarSpace := numBars*bc.barWidth + (numBars-1)*bc.barGap

	// scale if needed to fit
	barWidth := bc.barWidth
	barGap := bc.barGap
	if totalBarSpace > width {
		// reduce bar width and gap to fit
		availPerBar := width / numBars
		barWidth = max(1, availPerBar-1)
		barGap = max(0, availPerBar-barWidth)
		totalBarSpace = numBars*barWidth + (numBars-1)*barGap
	}

	// calculate padding to center
	leftPad := (width - totalBarSpace) / 2

	var sections []string
	// value labels above bars
	if bc.showValues {
		valueRow := strings.Repeat(" ", leftPad)
		for i, d := range bc.data {
			valueStr := bc.formatter(d.Value)
			// center the value above the bar
			padded := centerString(valueStr, barWidth)
			valueRow += padded
			if i < numBars-1 {
				valueRow += strings.Repeat(" ", barGap)
			}
		}
		valueStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)
		sections = append(sections, valueStyle.Render(valueRow))
	}

	// render bars
	// each bar is rendered separately, then combined horizontally
	barStrings := make([][]string, numBars)
	for i, d := range bc.data {
		fillPct := d.Value / bc.maxValue
		if fillPct > 1 {
			fillPct = 1
		}
		if fillPct < 0 {
			fillPct = 0
		}

		bar := NewSimpleBar(fillPct, bc.colorFunc(d.Value),
			WithBarHeight(bc.height),
			WithBarWidth(barWidth),
			WithBarBgColor(bc.bgColor),
		)
		barStrings[i] = strings.Split(bar.Render(), "\n")
	}

	// combine bars horizontally for each row
	for row := range bc.height {
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
	if bc.showAxis {
		labels := make([]string, numBars)
		for i, d := range bc.data {
			labels[i] = d.Label
		}
		axis := NewAxis(labels, totalBarSpace, WithAxisTextColor(theme.ColorDim))
		axisStr := axis.Render()

		// pad axis to match bar centering
		axisLines := strings.Split(axisStr, "\n")
		for _, line := range axisLines {
			sections = append(sections, strings.Repeat(" ", leftPad)+line)
		}
	}

	return strings.Join(sections, "\n")
}

// centerString centers a string within the given width.
func centerString(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	leftPad := (width - len(s)) / 2
	rightPad := width - len(s) - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}
