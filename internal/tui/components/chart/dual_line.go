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
	series1        []DataPoint
	series2        []DataPoint
	series1Lbl     string
	series2Lbl     string
	color1         color.Color
	color2         color.Color
	height         int
	minValue       float64
	maxValue       float64
	autoScale      bool // auto-scale min/max from data
	formatter      ValueFormatter
	showAxis       bool
	showLegend     bool
	legendPosition LegendPosition
	showDots       bool
	showValues     bool // show value labels above data points
	smoothness     int
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

// WithDualLineFormatter sets the value formatter.
func WithDualLineFormatter(f ValueFormatter) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.formatter = f
	}
}

// WithDualLineShowLegend shows the legend.
func WithDualLineShowLegend(show bool) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.showLegend = show
	}
}

// WithDualLineAutoScale enables auto-scaling min/max from data with padding.
func WithDualLineAutoScale(auto bool) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.autoScale = auto
	}
}

// WithDualLineShowValues shows value labels above data points.
func WithDualLineShowValues(show bool) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.showValues = show
	}
}

// WithDualLineLegendPosition sets the legend position.
func WithDualLineLegendPosition(pos LegendPosition) DualLineChartOption {
	return func(dlc *DualLineChart) {
		dlc.legendPosition = pos
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
		showValues: false,
		smoothness: 8,
		autoScale:  false,
	}
	for _, opt := range opts {
		opt(dlc)
	}

	dlc.computeMinMax()
	return dlc
}

// computeMinMax auto-detects min/max from data if needed.
func (dlc *DualLineChart) computeMinMax() {
	if !dlc.autoScale && dlc.maxValue > dlc.minValue {
		return
	}

	dataMin, dataMax := findDataMinMax(dlc.series1, dlc.series2)

	if dataMax <= dataMin {
		dlc.minValue = 0
		dlc.maxValue = 100
		return
	}

	padding := (dataMax - dataMin) * 0.1
	if dlc.autoScale {
		dlc.minValue = dataMin - padding
		dlc.maxValue = dataMax + padding
	} else {
		dlc.maxValue = dataMax + padding
	}
}

// findDataMinMax finds the min and max values across multiple data series.
func findDataMinMax(series ...[]DataPoint) (dataMin, dataMax float64) {
	dataMin = 1e18
	dataMax = -1e18
	for _, s := range series {
		for _, d := range s {
			if d.Value < dataMin {
				dataMin = d.Value
			}
			if d.Value > dataMax {
				dataMax = d.Value
			}
		}
	}
	return dataMin, dataMax
}

// Render renders the dual line chart.
func (dlc *DualLineChart) Render(width int) string {
	if (len(dlc.series1) == 0 && len(dlc.series2) == 0) || width <= 0 {
		return ""
	}

	var sections []string

	if dlc.showLegend && dlc.legendPosition == LegendTopRight {
		items := []LegendItem{
			{Label: dlc.series1Lbl, Color: dlc.color1},
			{Label: dlc.series2Lbl, Color: dlc.color2},
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

	// determine the number of points (use the longer series)
	numPoints := max(len(dlc.series1), len(dlc.series2))

	// braille canvas dimensions
	dotsWidth := width * 2
	dotsHeight := dlc.height * 4

	// create canvas for series 1
	canvas1 := drawille.NewCanvas()
	normalized1 := normalizeDataPoints(dlc.series1, dlc.minValue, dlc.maxValue)
	drawLineOnCanvas(&canvas1, normalized1, dotsWidth, dotsHeight, numPoints, dlc.showDots, dlc.smoothness)

	// create canvas for series 2
	canvas2 := drawille.NewCanvas()
	normalized2 := normalizeDataPoints(dlc.series2, dlc.minValue, dlc.maxValue)
	drawLineOnCanvas(&canvas2, normalized2, dotsWidth, dotsHeight, numPoints, dlc.showDots, dlc.smoothness)

	// render both canvases and combine with colors
	str1 := getCanvasString(&canvas1, dotsWidth, dotsHeight)
	str2 := getCanvasString(&canvas2, dotsWidth, dotsHeight)

	// value labels above the chart (two rows - one for each series)
	if dlc.showValues {
		label1Row := dlc.renderValueRow(dlc.series1, width, numPoints, dlc.color1)
		label2Row := dlc.renderValueRow(dlc.series2, width, numPoints, dlc.color2)
		sections = append(sections, label1Row)
		sections = append(sections, label2Row)
	}

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

	if dlc.showLegend && dlc.legendPosition == LegendBottom {
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

// renderValueRow renders a single row of value labels for a series.
func (dlc *DualLineChart) renderValueRow(data []DataPoint, width, numPoints int, clr color.Color) string {
	if len(data) == 0 {
		return strings.Repeat(" ", width)
	}

	// calculate X positions matching the line renderer
	dotsWidth := width * 2
	var xStep float64
	if numPoints == 1 {
		xStep = 0
	} else {
		xStep = float64(dotsWidth-1) / float64(numPoints-1)
	}

	row := make([]rune, width)
	for i := range row {
		row[i] = ' '
	}

	for i, d := range data {
		valueStr := dlc.formatter(d.Value)

		dotX := int(float64(i) * xStep)
		charX := dotX / 2

		// center the label around charX
		startX := charX - len(valueStr)/2
		if startX < 0 {
			startX = 0
		}
		if startX+len(valueStr) > width {
			startX = width - len(valueStr)
		}

		for j, r := range valueStr {
			if startX+j >= 0 && startX+j < width {
				row[startX+j] = r
			}
		}
	}

	style := lipgloss.NewStyle().Foreground(clr)
	return style.Render(string(row))
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
func drawLineOnCanvas(canvas *drawille.Canvas, points []float64, dotsWidth, dotsHeight, totalPoints int, showDots bool, smoothness int) {
	if len(points) == 0 {
		return
	}

	// calculate x step based on total points (not just this series)
	xStep := float64(dotsWidth-1) / float64(totalPoints-1)
	if totalPoints <= 1 {
		xStep = 0
	}

	// generate interpolated points for smooth curves
	var interpolatedPoints [][2]float64
	if smoothness > 0 && len(points) > 1 {
		interpolatedPoints = interpolateLinePoints(points, dotsHeight, xStep, smoothness)
	} else {
		// no interpolation, just use original points
		for i, p := range points {
			x := float64(i) * xStep
			y := (1 - p) * float64(dotsHeight-1)
			interpolatedPoints = append(interpolatedPoints, [2]float64{x, y})
		}
	}

	// draw lines between interpolated points
	for i := 0; i < len(interpolatedPoints)-1; i++ {
		x1 := int(interpolatedPoints[i][0])
		y1 := int(interpolatedPoints[i][1])
		x2 := int(interpolatedPoints[i+1][0])
		y2 := int(interpolatedPoints[i+1][1])

		drawBresenhamLine(canvas, x1, y1, x2, y2, 1)
	}

	// draw dots at original data points
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

// interpolateLinePoints generates smooth curve points using Catmull-Rom spline interpolation.
func interpolateLinePoints(points []float64, dotsHeight int, xStep float64, smoothness int) [][2]float64 {
	n := len(points)
	var result [][2]float64

	for i := 0; i < n-1; i++ {
		// get 4 control points for Catmull-Rom (p0, p1, p2, p3)
		p0 := points[max(0, i-1)]
		p1 := points[i]
		p2 := points[i+1]
		p3 := points[min(n-1, i+2)]

		x0 := float64(max(0, i-1)) * xStep
		x1 := float64(i) * xStep
		x2 := float64(i+1) * xStep
		x3 := float64(min(n-1, i+2)) * xStep

		// convert y values to canvas coordinates
		y0 := (1 - p0) * float64(dotsHeight-1)
		y1 := (1 - p1) * float64(dotsHeight-1)
		y2 := (1 - p2) * float64(dotsHeight-1)
		y3 := (1 - p3) * float64(dotsHeight-1)

		// generate interpolated points along this segment
		for j := 0; j <= smoothness; j++ {
			t := float64(j) / float64(smoothness)

			// catmull-Rom spline formula
			x := catmullRom(x0, x1, x2, x3, t)
			y := catmullRom(y0, y1, y2, y3, t)

			// clamp y to valid range
			if y < 0 {
				y = 0
			}
			if y > float64(dotsHeight-1) {
				y = float64(dotsHeight - 1)
			}

			result = append(result, [2]float64{x, y})
		}
	}

	return result
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
			r1, r2 := ' ', ' '
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
