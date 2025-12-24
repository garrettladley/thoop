package gauge

import (
	"math"

	drawille "github.com/exrook/drawille-go"
)

const (
	// arc parameters (degrees)
	// screen coords: 0°=right(3 o'clock), 90°=down(6 o'clock), 180°=left(9 o'clock), 270°=up(12 o'clock)
	// start at top (12 o'clock = 270°), fill clockwise (increasing angle)
	arcStartAngle = 270.0 // top of circle (12 o'clock)
	arcSweep      = 360.0
	arcThickness  = 5
)

// drawArc draws a thick arc on the canvas from startAngle sweeping through sweepAngle degrees.
// uses the midpoint circle algorithm for clean, gap-free rendering.
// see: https://en.wikipedia.org/wiki/Midpoint_circle_algorithm
func drawArc(canvas *drawille.Canvas, centerX, centerY, radius float64, startAngle, sweepAngle float64) {
	endAngle := startAngle + sweepAngle

	for t := range arcThickness {
		r := int(radius) - t
		if r <= 0 {
			continue
		}
		midpointCircleArc(canvas, int(centerX), int(centerY), r, startAngle, endAngle)
	}
}

// midpointCircleArc draws an arc using the midpoint circle algorithm.
// we use integer arithmetic to avoid floating-point gaps.
func midpointCircleArc(canvas *drawille.Canvas, cx, cy, radius int, startAngle, endAngle float64) {
	x := radius
	y := 0
	d := 1 - radius // decision parameter

	for x >= y {
		// draw the 8 symmetric points, filtered by angle
		drawOctantPoints(canvas, cx, cy, x, y, startAngle, endAngle)

		y++
		if d < 0 {
			d += 2*y + 1
		} else {
			x--
			d += 2*(y-x) + 1
		}
	}
}

// drawOctantPoints draws points in all 8 octants that fall within the angle range.
func drawOctantPoints(canvas *drawille.Canvas, cx, cy, x, y int, startAngle, endAngle float64) {
	// the 8 symmetric points of a circle
	points := [][2]int{
		{cx + x, cy - y}, // Octant 1: 0° to 45° (top-right, above diagonal)
		{cx + y, cy - x}, // Octant 2: 45° to 90° (top-right, below diagonal)
		{cx - y, cy - x}, // Octant 3: 90° to 135° (top-left, below diagonal)
		{cx - x, cy - y}, // Octant 4: 135° to 180° (top-left, above diagonal)
		{cx - x, cy + y}, // Octant 5: 180° to 225° (bottom-left, above diagonal)
		{cx - y, cy + x}, // Octant 6: 225° to 270° (bottom-left, below diagonal)
		{cx + y, cy + x}, // Octant 7: 270° to 315° (bottom-right, below diagonal)
		{cx + x, cy + y}, // Octant 8: 315° to 360° (bottom-right, above diagonal)
	}

	for _, p := range points {
		if isInArcRange(cx, cy, p[0], p[1], startAngle, endAngle) {
			canvas.Set(p[0], p[1])
		}
	}
}

// isInArcRange checks if a point's angle from center falls within [startAngle, endAngle].
// handles wraparound correctly (e.g., startAngle=345, endAngle=675 for a 330° arc).
func isInArcRange(cx, cy, px, py int, startAngle, endAngle float64) bool {
	// calculate angle of point from center
	// in screen coords, Y increases downward, so we use (py-cy) directly
	dx := float64(px - cx)
	dy := float64(py - cy)

	// atan2 returns angle in radians from -π to π
	angle := math.Atan2(dy, dx) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}

	// normalize endAngle for comparison
	// if arc wraps around 360° (e.g., 345° to 675°), we need special handling
	if endAngle > 360 {
		// arc wraps around: check if angle is >= start OR <= (end - 360)
		if angle >= startAngle || angle <= (endAngle-360) {
			return true
		}
	} else {
		// normal case: check if angle is within [start, end]
		if angle >= startAngle && angle <= endAngle {
			return true
		}
	}

	return false
}

func drawFullArc(canvas *drawille.Canvas, centerX, centerY, radius float64) {
	drawArc(canvas, centerX, centerY, radius, arcStartAngle, arcSweep)
}

func drawFilledArc(canvas *drawille.Canvas, centerX, centerY, radius float64, fillPercent float64) {
	if fillPercent <= 0 {
		return
	}
	if fillPercent > 1 {
		fillPercent = 1
	}
	sweepAngle := fillPercent * arcSweep
	drawArc(canvas, centerX, centerY, radius, arcStartAngle, sweepAngle)
}
