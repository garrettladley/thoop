package chart

import (
	"image/color"
	"math"
	"strings"

	drawille "github.com/exrook/drawille-go"

	"charm.land/lipgloss/v2"
)

// Line renders a line using braille characters.
type Line struct {
	points     []float64   // normalized values [0, 1]
	width      int         // width in characters
	height     int         // height in characters
	color      color.Color // line color
	showDots   bool        // show dots at data points
	lineWidth  int         // line thickness in braille dots
	smoothness int         // interpolation points between data points (0 = straight lines)
}

// LineOption configures a Line.
type LineOption func(*Line)

// WithLineColor sets the line color.
func WithLineColor(c color.Color) LineOption {
	return func(l *Line) {
		l.color = c
	}
}

// WithLineHeight sets the line height in characters.
func WithLineHeight(h int) LineOption {
	return func(l *Line) {
		l.height = h
	}
}

// WithLineShowDots shows dots at data points.
func WithLineShowDots(show bool) LineOption {
	return func(l *Line) {
		l.showDots = show
	}
}

// WithLineWidth sets the line thickness.
func WithLineWidth(w int) LineOption {
	return func(l *Line) {
		l.lineWidth = w
	}
}

// WithLineSmoothness sets the number of interpolation points between data points.
// Higher values create smoother curves. 0 = straight lines (default: 8).
func WithLineSmoothness(s int) LineOption {
	return func(l *Line) {
		l.smoothness = s
	}
}

// NewLine creates a new line renderer.
func NewLine(points []float64, width int, opts ...LineOption) *Line {
	l := &Line{
		points:     points,
		width:      width,
		height:     6,
		showDots:   true,
		lineWidth:  1,
		smoothness: 8, // default interpolation for smooth curves
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Render renders the line using braille characters.
func (l *Line) Render() string {
	result := l.RenderUncolored()

	// apply color
	if l.color != nil {
		style := lipgloss.NewStyle().Foreground(l.color)
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			lines[i] = style.Render(line)
		}
		result = strings.Join(lines, "\n")
	}

	return result
}

// RenderUncolored renders the line without color styling.
func (l *Line) RenderUncolored() string {
	if len(l.points) == 0 || l.width <= 0 || l.height <= 0 {
		return ""
	}

	// braille canvas: 2 dots per char width, 4 dots per char height
	dotsWidth := l.width * 2
	dotsHeight := l.height * 4

	canvas := drawille.NewCanvas()

	// Calculate x positions for each point
	xStep := float64(dotsWidth-1) / float64(len(l.points)-1)
	if len(l.points) == 1 {
		xStep = 0
	}

	// generate interpolated points for smooth curves
	var interpolatedPoints [][2]float64
	if l.smoothness > 0 && len(l.points) > 1 {
		interpolatedPoints = l.interpolatePoints(dotsWidth, dotsHeight, xStep)
	} else {
		// no interpolation, just use original points
		for i, p := range l.points {
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

		drawBresenhamLine(&canvas, x1, y1, x2, y2, l.lineWidth)
	}

	// draw dots at original data points if enabled
	if l.showDots {
		for i, p := range l.points {
			x := int(float64(i) * xStep)
			y := int((1 - p) * float64(dotsHeight-1))
			// draw a small dot (3x3 braille dots)
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

	return getCanvasString(&canvas, dotsWidth, dotsHeight)
}

// interpolatePoints generates smooth curve points using Catmull-Rom spline interpolation.
func (l *Line) interpolatePoints(_, dotsHeight int, xStep float64) [][2]float64 {
	points := l.points
	n := len(points)
	var result [][2]float64

	for i := 0; i < n-1; i++ {
		// get 4 control points for Catmull-Rom (p0, p1, p2, p3)
		// p1 and p2 are the segment endpoints
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
		for j := 0; j <= l.smoothness; j++ {
			t := float64(j) / float64(l.smoothness)

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

// catmullRom computes a point on a Catmull-Rom spline.
func catmullRom(p0, p1, p2, p3, t float64) float64 {
	t2 := t * t
	t3 := t2 * t
	return 0.5 * ((2 * p1) +
		(-p0+p2)*t +
		(2*p0-5*p1+4*p2-p3)*t2 +
		(-p0+3*p1-3*p2+p3)*t3)
}

// drawBresenhamLine draws a line using Bresenham's algorithm.
func drawBresenhamLine(canvas *drawille.Canvas, x1, y1, x2, y2, thickness int) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	if x1 >= x2 {
		sx = -1
	}
	sy := 1
	if y1 >= y2 {
		sy = -1
	}
	err := dx - dy

	for {
		// set pixel with thickness
		for tx := -thickness / 2; tx <= thickness/2; tx++ {
			for ty := -thickness / 2; ty <= thickness/2; ty++ {
				canvas.Set(x1+tx, y1+ty)
			}
		}

		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// getCanvasString extracts the canvas as a string with consistent dimensions.
func getCanvasString(canvas *drawille.Canvas, width, height int) string {
	charWidth := width / 2
	charHeight := height / 4

	rows := canvas.Rows(0, 0, width, height)

	var lines []string
	for i := range charHeight {
		if i < len(rows) {
			line := rows[i]
			runeCount := len([]rune(line))
			if runeCount < charWidth {
				line += strings.Repeat(" ", charWidth-runeCount)
			} else if runeCount > charWidth {
				line = string([]rune(line)[:charWidth])
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, strings.Repeat(" ", charWidth))
		}
	}

	return strings.Join(lines, "\n")
}

// NormalizeValues normalizes a slice of values to [0, 1] range.
func NormalizeValues(values []float64, minVal, maxVal float64) []float64 {
	if len(values) == 0 {
		return nil
	}

	// auto-detect min/max if not provided
	if minVal == maxVal {
		minVal = math.MaxFloat64
		maxVal = -math.MaxFloat64
		for _, v := range values {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
	}

	if minVal == maxVal {
		// all values are the same
		result := make([]float64, len(values))
		for i := range result {
			result[i] = 0.5
		}
		return result
	}

	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = (v - minVal) / (maxVal - minVal)
		if result[i] < 0 {
			result[i] = 0
		}
		if result[i] > 1 {
			result[i] = 1
		}
	}
	return result
}
