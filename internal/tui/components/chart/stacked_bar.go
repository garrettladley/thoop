package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// LegendPosition specifies where the legend is rendered.
type LegendPosition int

const (
	LegendBottom LegendPosition = iota
	LegendTopRight
)

// StackedBarChart renders a stacked vertical bar chart.
type StackedBarChart struct {
	data           []StackedDataPoint
	height         int
	colors         []color.Color
	labels         []string // legend labels for each segment
	formatter      ValueFormatter
	showValues     bool
	showAxis       bool
	showLegend     bool
	legendPosition LegendPosition
	barWidth       int
	barGap         int
	bgColor        color.Color
	showSegTotal   bool // show total value above bars
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

// WithStackedBarShowLegend shows the legend.
func WithStackedBarShowLegend(show bool) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.showLegend = show
	}
}

// WithStackedBarLegendPosition sets the legend position.
func WithStackedBarLegendPosition(pos LegendPosition) StackedBarChartOption {
	return func(sbc *StackedBarChart) {
		sbc.legendPosition = pos
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

	// calculate bar dimensions to fill width
	numBars := len(sbc.data)

	// distribute width evenly among bars with gaps
	// we want gap to be roughly 30% of available space per bar
	availPerBar := float64(width) / float64(numBars)
	barWidth := max(
		// 70% for bar
		int(availPerBar*0.7), 2)
	barGap := max(
		// remaining for gap
		int(availPerBar)-barWidth, 1)
	totalBarSpace := numBars*barWidth + (numBars-1)*barGap

	// calculate padding to center (should be minimal now)
	leftPad := max((width-totalBarSpace)/2, 0)

	// calculate totals and fill heights for each bar
	totals := make([]float64, numBars)
	fillHeights := make([]int, numBars)
	for i, d := range sbc.data {
		total := 0.0
		for _, v := range d.Values {
			total += v
		}
		totals[i] = total
		fillPct := total / maxTotal
		if fillPct > 1 {
			fillPct = 1
		}
		if fillPct > 0 {
			fillHeights[i] = max(1, int(fillPct*float64(sbc.height)+0.5))
		}
	}

	sections := make([]string, 0, sbc.height+6)

	// render legend at top right if configured
	if sbc.showLegend && len(sbc.labels) > 0 && sbc.legendPosition == LegendTopRight {
		var items []LegendItem
		for i, label := range sbc.labels {
			c := sbc.bgColor
			if i < len(sbc.colors) {
				c = sbc.colors[i]
			}
			items = append(items, LegendItem{Label: label, Color: c})
		}
		legend := NewLegend(items)
		legendStr := legend.Render()
		legendWidth := lipgloss.Width(legendStr)
		padding := width - legendWidth
		if padding > 0 {
			sections = append(sections, strings.Repeat(" ", padding)+legendStr)
		} else {
			sections = append(sections, legendStr)
		}
		sections = append(sections, "")
	}

	// render stacked bars
	barStrings := make([][]string, numBars)
	for i, d := range sbc.data {
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
		)
		barStrings[i] = strings.Split(bar.Render(), "\n")
	}

	// calculate the row where each bar starts (from top) - this is where the label goes above
	labelRows := make([]int, numBars)
	for i := range numBars {
		emptyRows := sbc.height - fillHeights[i]
		if emptyRows > 0 {
			labelRows[i] = emptyRows - 1
		} else {
			labelRows[i] = -1 // label above the chart area
		}
	}

	// determine if we need a label row above the chart (for bars that fill the entire height)
	needsTopLabelRow := false
	if sbc.showValues {
		for i := range numBars {
			if labelRows[i] < 0 {
				needsTopLabelRow = true
				break
			}
		}
	}

	// render top label row if needed (for full-height bars)
	if sbc.showValues && needsTopLabelRow {
		var rowStr strings.Builder
		rowStr.WriteString(strings.Repeat(" ", leftPad))
		for i := range numBars {
			if labelRows[i] < 0 {
				valueStr := sbc.formatter(totals[i])
				padded := centerString(valueStr, barWidth)
				rowStr.WriteString(lipgloss.NewStyle().Foreground(theme.ColorWhite).Render(padded))
			} else {
				rowStr.WriteString(strings.Repeat(" ", barWidth))
			}
			if i < numBars-1 {
				rowStr.WriteString(strings.Repeat(" ", barGap))
			}
		}
		sections = append(sections, rowStr.String())
	}

	// combine bars horizontally, inserting labels at the right positions
	for row := range sbc.height {
		var rowStr strings.Builder
		rowStr.WriteString(strings.Repeat(" ", leftPad))
		for i := range numBars {
			// check if this row should show a label for this bar
			if sbc.showValues && labelRows[i] == row {
				valueStr := sbc.formatter(totals[i])
				padded := centerString(valueStr, barWidth)
				rowStr.WriteString(lipgloss.NewStyle().Foreground(theme.ColorWhite).Render(padded))
			} else if row < len(barStrings[i]) {
				rowStr.WriteString(barStrings[i][row])
			} else {
				rowStr.WriteString(strings.Repeat(" ", barWidth))
			}
			if i < numBars-1 {
				rowStr.WriteString(strings.Repeat(" ", barGap))
			}
		}
		sections = append(sections, rowStr.String())
	}

	// X-axis labels
	if sbc.showAxis {
		labels := make([]string, numBars)
		positions := make([]int, numBars)
		for i, d := range sbc.data {
			labels[i] = d.Label
			positions[i] = i*(barWidth+barGap) + barWidth/2
		}
		axis := NewAxis(labels, totalBarSpace, WithAxisTextColor(theme.ColorDim), WithAxisPositions(positions))
		axisStr := axis.Render()
		axisLines := strings.SplitSeq(axisStr, "\n")
		for line := range axisLines {
			sections = append(sections, strings.Repeat(" ", leftPad)+line)
		}
	}

	// render legend at bottom if configured (default)
	if sbc.showLegend && len(sbc.labels) > 0 && sbc.legendPosition == LegendBottom {
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
