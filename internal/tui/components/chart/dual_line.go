package chart

import (
	"image/color"
	"strings"

	drawille "github.com/exrook/drawille-go"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/braille"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

// DualLineChart renders two line series overlaid on the same chart.
type DualLineChart struct {
	series1    []DataPoint
	series2    []DataPoint
	series1Lbl string
	series2Lbl string
	color1     color.Color
	color2     color.Color
	height     int
	minValue   float64
	maxValue   float64
	formatter  ValueFormatter
	showAxis   bool
	showLegend bool
	showDots   bool
}

// DualLineChartOption configures a DualLineChart.
type DualLineChartOption func(*DualLineChart)

// WithDualLineHeight sets the chart height.
func WithDualLineHeight(h int) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.height = h
	}
}

// WithDualLineColors sets the colors for both series.
func WithDualLineColors(c1, c2 color.Color) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.color1 = c1
		dlc.color2 = c2
	}
}

// WithDualLineLabels sets the labels for both series.
func WithDualLineLabels(l1, l2 string) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.series1Lbl = l1
		dlc.series2Lbl = l2
	}
}

// WithDualLineMin sets the minimum value for scaling.
func WithDualLineMin(min float64) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.minValue = min
	}
}

// WithDualLineMax sets the maximum value for scaling.
func WithDualLineMax(max float64) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.maxValue = max
	}
}

// WithDualLineFormatter sets the value formatter.
func WithDualLineFormatter(f ValueFormatter) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.formatter = f
	}
}

// WithDualLineShowAxis shows the x-axis labels.
func WithDualLineShowAxis(show bool) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.showAxis = show
	}
}

// WithDualLineShowLegend shows the legend.
func WithDualLineShowLegend(show bool) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.showLegend = show
	}
}

// WithDualLineShowDots shows dots at data points.
func WithDualLineShowDots(show bool) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.showDots = show
	}
}

// NewDualLineChart creates a new dual line chart.
func NewDualLineChart(series1, series2 []DataPoint, opts ...DualLineChartOption) *DualLineChart {
	dlc := &DualLineChart{
		series1:    series1,
		series2:    series2,
		series1Lbl: "Series 1",
		series2Lbl: "Series 2",
		color1:     theme.ColorSleep,
		color2:     theme.ColorDim,
		height:     6,
		formatter:  FormatInt,
		showAxis:   true,
		showLegend: true,
		showDots:   true,
	}
	for _, opt := range opts {
		opt(dlc)
	}

	// Auto-detect min/max
	if dlc.maxValue <= dlc.minValue {
		for _, d := range dlc.series1 {
			if d.Value > dlc.maxValue {
				dlc.maxValue = d.Value
			}
		}
		for _, d := range dlc.series2 {
			if d.Value > dlc.maxValue {
				dlc.maxValue = d.Value
			}
		}
		if dlc.maxValue <= 0 {
			dlc.maxValue = 100
		}
	}

	return dlc
}

// Render renders the dual line chart.
func (dlc *DualLineChart) Render(width int) string {
	if (len(dlc.series1) == 0 && len(dlc.series2) == 0) || width <= 0 {
		return ""
	}

	var sections []string

	// determine the number of points (use the longer series)
	numPoints := max(len(dlc.series1), len(dlc.series2))

	// braille canvas dimensions
	dotsWidth := width * 2
	dotsHeight := dlc.height * 4

	// create canvas for series 1
	canvas1 := drawille.NewCanvas()
	normalized1 := normalizeDataPoints(dlc.series1, dlc.minValue, dlc.maxValue)
	drawLineOnCanvas(&canvas1, normalized1, dotsWidth, dotsHeight, numPoints, dlc.showDots)

	// create canvas for series 2
	canvas2 := drawille.NewCanvas()
	normalized2 := normalizeDataPoints(dlc.series2, dlc.minValue, dlc.maxValue)
	drawLineOnCanvas(&canvas2, normalized2, dotsWidth, dotsHeight, numPoints, dlc.showDots)

	// render both canvases and combine with colors
	str1 := getCanvasString(&canvas1, dotsWidth, dotsHeight)
	str2 := getCanvasString(&canvas2, dotsWidth, dotsHeight)

	combined := overlayLines(str1, str2, dlc.color1, dlc.color2)
	sections = append(sections, combined)

	// X-axis labels
	if dlc.showAxis && numPoints > 0 {
		// use labels from series1 (or series2 if series1 is empty)
		series := dlc.series1
		if len(series) == 0 {
			series = dlc.series2
		}
		labels := make([]string, len(series))
		for i, d := range series {
			labels[i] = d.Label
		}
		axis := NewAxis(labels, width, WithAxisTextColor(theme.ColorDim))
		sections = append(sections, axis.Render())
	}

	// legend
	if dlc.showLegend {
		items := []LegendItem{
			{Label: dlc.series1Lbl, Color: dlc.color1},
			{Label: dlc.series2Lbl, Color: dlc.color2},
		}
		legend := NewLegend(items)
		sections = append(sections, "")
		sections = append(sections, legend.Render())
	}

	return strings.Join(sections, "\n")
}

// normalizeDataPoints normalizes data points to [0, 1] range.
func normalizeDataPoints(data []DataPoint, minVal, maxVal float64) []float64 {
	result := make([]float64, len(data))
	for i, d := range data {
		if maxVal > minVal {
			result[i] = (d.Value - minVal) / (maxVal - minVal)
		} else {
			result[i] = 0.5
		}
		if result[i] < 0 {
			result[i] = 0
		}
		if result[i] > 1 {
			result[i] = 1
		}
	}
	return result
}

// drawLineOnCanvas draws a line on a braille canvas.
func drawLineOnCanvas(canvas *drawille.Canvas, points []float64, dotsWidth, dotsHeight, totalPoints int, showDots bool) {
	if len(points) == 0 {
		return
	}

	// calculate x step based on total points (not just this series)
	xStep := float64(dotsWidth-1) / float64(totalPoints-1)
	if totalPoints <= 1 {
		xStep = 0
	}

	// draw lines
	for i := 0; i < len(points)-1; i++ {
		x1 := int(float64(i) * xStep)
		y1 := int((1 - points[i]) * float64(dotsHeight-1))
		x2 := int(float64(i+1) * xStep)
		y2 := int((1 - points[i+1]) * float64(dotsHeight-1))

		drawBresenhamLine(canvas, x1, y1, x2, y2, 1)
	}

	// draw dots
	if showDots {
		for i, p := range points {
			x := int(float64(i) * xStep)
			y := int((1 - p) * float64(dotsHeight-1))
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					px := x + dx
					py := y + dy
					if px >= 0 && px < dotsWidth && py >= 0 && py < dotsHeight {
						canvas.Set(px, py)
					}
				}
			}
		}
	}
}

const emptyBraille rune = '\u2800'

// overlayLines combines two line renders with different colors.
func overlayLines(str1, str2 string, color1, color2 color.Color) string {
	lines1 := strings.Split(str1, "\n")
	lines2 := strings.Split(str2, "\n")

	maxLines := max(len(lines1), len(lines2))
	result := make([]string, maxLines)

	style1 := lipgloss.NewStyle().Foreground(color1)
	style2 := lipgloss.NewStyle().Foreground(color2)

	for i := range maxLines {
		var line1, line2 string
		if i < len(lines1) {
			line1 = lines1[i]
		}
		if i < len(lines2) {
			line2 = lines2[i]
		}

		runes1 := []rune(line1)
		runes2 := []rune(line2)

		maxLen := max(len(runes1), len(runes2))
		var lineBuilder strings.Builder

		for j := range maxLen {
			var r1, r2 rune = ' ', ' '
			if j < len(runes1) {
				r1 = runes1[j]
			}
			if j < len(runes2) {
				r2 = runes2[j]
			}

			b1 := braille.Is(r1) && r1 != emptyBraille
			b2 := braille.Is(r2) && r2 != emptyBraille

			if b1 && b2 {
				// both have content - show series1 on top
				lineBuilder.WriteString(style1.Render(string(r1)))
			} else if b1 {
				lineBuilder.WriteString(style1.Render(string(r1)))
			} else if b2 {
				lineBuilder.WriteString(style2.Render(string(r2)))
			} else {
				lineBuilder.WriteRune(' ')
			}
		}
		result[i] = lineBuilder.String()
	}

	return strings.Join(result, "\n")
}
