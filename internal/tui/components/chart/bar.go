package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// Block characters for sub-cell resolution (each represents 1/8 of a cell height).
// Index 0 = empty, Index 8 = full block.
var verticalBlocks = theme.VerticalBlocks

// Bar represents a single vertical bar with optional stacking.
type Bar struct {
	height   int       // height in terminal rows
	width    int       // width of the bar
	segments []Segment // segments from bottom to top
}

// Segment represents a colored segment of a bar.
type Segment struct {
	Value float64
	Color color.Color
}

// BarOption configures a Bar.
type BarOption func(*Bar)

// WithBarHeight sets the bar height in terminal rows.
func WithBarHeight(h int) BarOption {
	return func(b *Bar) {
		b.height = h
	}
}

// WithBarWidth sets the bar width in characters.
func WithBarWidth(w int) BarOption {
	return func(b *Bar) {
		b.width = w
	}
}

// WithBarBgColor sets the background color.

// NewBar creates a new bar with the given segments.
// Values should be normalized to [0, 1] representing the fill percentage.
func NewBar(segments []Segment, opts ...BarOption) *Bar {
	b := &Bar{
		height:   8,
		width:    3,
		segments: segments,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// NewSimpleBar creates a bar with a single segment.
func NewSimpleBar(value float64, c color.Color, opts ...BarOption) *Bar {
	return NewBar([]Segment{{Value: value, Color: c}}, opts...)
}

// Render renders the bar as a string.
func (b *Bar) Render() string {
	if b.height <= 0 || b.width <= 0 {
		return ""
	}

	totalUnits := b.height * 8

	lines := make([]string, b.height)

	type segmentRange struct {
		startUnit int
		endUnit   int
		color     color.Color
	}
	var ranges []segmentRange

	currentUnit := 0
	for _, seg := range b.segments {
		units := int(seg.Value * float64(totalUnits))
		if units > 0 {
			ranges = append(ranges, segmentRange{
				startUnit: currentUnit,
				endUnit:   currentUnit + units,
				color:     seg.Color,
			})
			currentUnit += units
		}
	}

	// calculate total fill height for proper stacking
	totalFillUnits := 0
	for _, r := range ranges {
		totalFillUnits = max(totalFillUnits, r.endUnit)
	}

	// render each row (from top to bottom in output, but bottom to top in value)
	for row := range b.height {
		// this row represents units from rowStartUnit to rowEndUnit
		rowStartUnit := (b.height - 1 - row) * 8
		rowEndUnit := rowStartUnit + 8

		var rowContent strings.Builder
		rowChar := ' '
		var rowColor color.Color

		// check how much of the total stack fills this row
		if totalFillUnits > rowStartUnit {
			fillInRow := min(totalFillUnits, rowEndUnit) - rowStartUnit
			if fillInRow >= 8 {
				rowChar = theme.VerticalBlocks[8]
			} else if fillInRow > 0 {
				rowChar = verticalBlocks[fillInRow]
			}

			// find the color: use the topmost segment that overlaps this row
			for i := len(ranges) - 1; i >= 0; i-- {
				r := ranges[i]
				if r.endUnit > rowStartUnit && r.startUnit < rowEndUnit {
					rowColor = r.color
					break
				}
			}
		}

		// build the row string
		for range b.width {
			rowContent.WriteRune(rowChar)
		}

		style := lipgloss.NewStyle()
		if rowColor != nil {
			style = style.Foreground(rowColor)
		}

		lines[row] = style.Render(rowContent.String())
	}

	return strings.Join(lines, "\n")
}
