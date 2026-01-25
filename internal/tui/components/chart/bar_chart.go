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
	minValue   float64
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

// WithBarChartMin sets the minimum value for scaling.
func WithBarChartMin(min float64) BarChartOption {
	return func(bc *BarChart) {
		bc.minValue = min
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
		barWidth:   4,
		barGap:     2,
		bgColor:    nil,
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

	// calculate bar dimensions to fill width
	numBars := len(bc.data)

	// distribute width evenly among bars with gaps
	// total space = numBars * barWidth + (numBars-1) * barGap = width
	// we want gap to be roughly 1/3 of bar width
	// so: numBars * barWidth + (numBars-1) * (barWidth/3) = width
	// solving: barWidth * (numBars + (numBars-1)/3) = width
	// barWidth = width / (numBars + (numBars-1)/3)
	availPerBar := float64(width) / float64(numBars)
	barWidth := int(availPerBar * 0.7) // 70% for bar
	if barWidth < 2 {
		barWidth = 2
	}
	barGap := int(availPerBar) - barWidth // remaining for gap
	if barGap < 1 {
		barGap = 1
	}
	totalBarSpace := numBars*barWidth + (numBars-1)*barGap

	// calculate padding to center (should be minimal now)
	leftPad := (width - totalBarSpace) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	// calculate fill percentages and label rows for each bar
	fillPcts := make([]float64, numBars)
	labelRows := make([]int, numBars)
	valueRange := bc.maxValue - bc.minValue
	if valueRange <= 0 {
		valueRange = 1 // prevent division by zero
	}
	needsTopLabelRow := false
	for i, d := range bc.data {
		fillPct := (d.Value - bc.minValue) / valueRange
		if fillPct > 1 {
			fillPct = 1
		}
		if fillPct < 0 {
			fillPct = 0
		}
		fillPcts[i] = fillPct
		// calculate fill height in rows
		fillHeight := 0
		if fillPct > 0 {
			fillHeight = max(1, int(fillPct*float64(bc.height)+0.5))
		}
		// label goes one row above the bar's fill (row just above where fill starts)
		// fill starts at row (height - fillHeight), so label at (height - fillHeight - 1)
		labelRows[i] = bc.height - fillHeight - 1
		if labelRows[i] < 0 {
			needsTopLabelRow = true
		}
	}

	sections := make([]string, 0, bc.height+4)

	// render bars
	barStrings := make([][]string, numBars)
	for i := range bc.data {
		bar := NewSimpleBar(fillPcts[i], bc.colorFunc(bc.data[i].Value),
			WithBarHeight(bc.height),
			WithBarWidth(barWidth),
			WithBarBgColor(bc.bgColor),
		)
		barStrings[i] = strings.Split(bar.Render(), "\n")
	}

	// render top label row if any bar fills entire height
	if bc.showValues && needsTopLabelRow {
		rowStr := strings.Repeat(" ", leftPad)
		for i, d := range bc.data {
			if labelRows[i] < 0 {
				valueStr := bc.formatter(d.Value)
				padded := centerString(valueStr, barWidth)
				rowStr += lipgloss.NewStyle().Foreground(theme.ColorWhite).Render(padded)
			} else {
				rowStr += strings.Repeat(" ", barWidth)
			}
			if i < numBars-1 {
				rowStr += strings.Repeat(" ", barGap)
			}
		}
		sections = append(sections, rowStr)
	}

	// combine bars horizontally for each row, with per-bar labels
	for row := range bc.height {
		rowStr := strings.Repeat(" ", leftPad)
		for i := range numBars {
			if bc.showValues && labelRows[i] == row {
				// render label on this row for this bar
				valueStr := bc.formatter(bc.data[i].Value)
				padded := centerString(valueStr, barWidth)
				rowStr += lipgloss.NewStyle().Foreground(theme.ColorWhite).Render(padded)
			} else if row < len(barStrings[i]) {
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
		positions := make([]int, numBars)
		for i, d := range bc.data {
			labels[i] = d.Label
			// bar center is at: i * (barWidth + barGap) + barWidth/2
			positions[i] = i*(barWidth+barGap) + barWidth/2
		}
		axis := NewAxis(labels, totalBarSpace, WithAxisTextColor(theme.ColorDim), WithAxisPositions(positions))
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
// if the string is longer than width, it is returned as-is (no truncation).
func centerString(s string, width int) string {
	strWidth := lipgloss.Width(s)
	if strWidth >= width {
		return s
	}
	leftPad := (width - strWidth) / 2
	rightPad := width - strWidth - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}
