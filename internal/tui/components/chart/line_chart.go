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

	lc.computeMinMax()
	return lc
}

// computeMinMax auto-detects min/max from data if needed.
func (lc *LineChart) computeMinMax() {
	if lc.maxValue > lc.minValue {
		return
	}

	for _, d := range lc.data {
		if d.Value > lc.maxValue {
			lc.maxValue = d.Value
		}
		if d.Value < lc.minValue || lc.minValue == 0 {
			lc.minValue = d.Value
		}
	}

	if lc.minValue != lc.maxValue {
		return
	}

	lc.minValue = 0
	if lc.maxValue == 0 {
		lc.maxValue = 100
	}
}

// Render renders the line chart.
func (lc *LineChart) Render(width int) string {
	if len(lc.data) == 0 || width <= 0 {
		return ""
	}

	// extract values and normalize
	values := make([]float64, len(lc.data))
	for i, d := range lc.data {
		values[i] = d.Value
	}
	normalized := NormalizeValues(values, lc.minValue, lc.maxValue)

	// render line (uncolored so we can overlay labels)
	line := NewLine(normalized, width,
		WithLineHeight(lc.height),
		WithLineShowDots(lc.showDots),
	)

	var chartStr string
	if lc.showValues {
		chartStr = lc.renderWithValueLabels(line.RenderUncolored(), width)
	} else {
		// just apply color directly
		chartStr = line.RenderUncolored()
		if lc.color != nil {
			style := lipgloss.NewStyle().Foreground(lc.color)
			lines := strings.Split(chartStr, "\n")
			for i, ln := range lines {
				lines[i] = style.Render(ln)
			}
			chartStr = strings.Join(lines, "\n")
		}
	}

	var sections []string
	sections = append(sections, chartStr)

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

// renderWithValueLabels renders value labels just above each data point.
func (lc *LineChart) renderWithValueLabels(brailleStr string, width int) string {
	brailleLines := strings.Split(brailleStr, "\n")
	chartHeight := len(brailleLines)

	// normalize values to get Y positions
	values := make([]float64, len(lc.data))
	for i, d := range lc.data {
		values[i] = d.Value
	}
	normalized := NormalizeValues(values, lc.minValue, lc.maxValue)

	// Calculate positions matching Line renderer
	dotsWidth := width * 2
	dotsHeight := lc.height * 4
	var xStep float64
	if len(lc.data) == 1 {
		xStep = 0
	} else {
		xStep = float64(dotsWidth-1) / float64(len(lc.data)-1)
	}

	// create label grid - same height as chart, labels go 1 row above data point
	// add 1 extra row at top for labels of highest points
	totalRows := chartHeight + 1
	labelGrid := make([][]rune, totalRows)
	for i := range labelGrid {
		labelGrid[i] = make([]rune, width)
	}

	// place each label just above its data point's Y position
	for i, d := range lc.data {
		valueStr := lc.formatter(d.Value)

		// X position (matches Line renderer)
		dotX := int(float64(i) * xStep)
		charX := dotX / 2

		// Y position - data point's row in the chart
		dotY := (1 - normalized[i]) * float64(dotsHeight-1)
		charY := int(dotY) / 4 // which row of the chart (0 = top)

		// label goes in the row above the data point
		// grid row 0 = above the chart, rows 1..chartHeight = chart rows
		// data point at charY (0-indexed in chart) maps to grid row charY+1
		// label goes at grid row charY (one above)
		labelRow := charY
		if labelRow < 0 {
			labelRow = 0
		}
		if labelRow >= totalRows {
			labelRow = totalRows - 1
		}

		startX := charX - len(valueStr)/2
		if startX < 0 {
			startX = 0
		}
		if startX+len(valueStr) > width {
			startX = width - len(valueStr)
		}

		// write label to grid
		for j, r := range valueStr {
			if startX+j >= 0 && startX+j < width {
				labelGrid[labelRow][startX+j] = r
			}
		}
	}

	// build output - combine labels with chart
	lineStyle := lipgloss.NewStyle().Foreground(lc.color)
	labelStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)

	result := make([]string, totalRows)
	for row := 0; row < totalRows; row++ {
		var rowBuilder strings.Builder

		// get braille for this row (row 0 is above chart, has no braille)
		var brailleRunes []rune
		if row > 0 && row-1 < len(brailleLines) {
			brailleRunes = []rune(brailleLines[row-1])
		}
		for len(brailleRunes) < width {
			brailleRunes = append(brailleRunes, ' ')
		}

		col := 0
		for col < width {
			if labelGrid[row][col] != 0 {
				// consecutive label characters
				start := col
				for col < width && labelGrid[row][col] != 0 {
					col++
				}
				labelText := string(labelGrid[row][start:col])
				rowBuilder.WriteString(labelStyle.Render(labelText))
			} else {
				// consecutive non-label characters
				start := col
				for col < width && labelGrid[row][col] == 0 {
					col++
				}
				if row == 0 {
					// row above chart - just spaces
					rowBuilder.WriteString(strings.Repeat(" ", col-start))
				} else {
					// chart row - braille with color
					rowBuilder.WriteString(lineStyle.Render(string(brailleRunes[start:col])))
				}
			}
		}
		result[row] = rowBuilder.String()
	}

	return strings.Join(result, "\n")
}
