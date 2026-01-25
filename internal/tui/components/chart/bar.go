package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Block characters for sub-cell resolution (each represents 1/8 of a cell height).
// Index 0 = empty, Index 8 = full block.
var verticalBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Bar represents a single vertical bar with optional stacking.
type Bar struct {
	height   int         // height in terminal rows
	width    int         // width of the bar
	segments []Segment   // segments from bottom to top
	bgColor  color.Color // background color for unfilled portion
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
func WithBarBgColor(c color.Color) BarOption {
	return func(b *Bar) {
		b.bgColor = c
	}
}

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

	// render each row (from top to bottom in output, but bottom to top in value)
	for row := range b.height {
		// this row represents units from rowStartUnit to rowEndUnit
		rowStartUnit := (b.height - 1 - row) * 8
		rowEndUnit := rowStartUnit + 8

		var (
			rowContent strings.Builder
			rowChar    = ' '
			rowColor   color.Color
		)

		for _, r := range ranges {
			if r.endUnit <= rowStartUnit {
				// segment is completely below this row
				continue
			}
			if r.startUnit >= rowEndUnit {
				// segment is completely above this row
				continue
			}

			// segment overlaps with this row
			overlapStart := max(r.startUnit, rowStartUnit)
			overlapEnd := min(r.endUnit, rowEndUnit)
			fillUnits := overlapEnd - overlapStart

			if fillUnits >= 8 {
				rowChar = '█'
			} else if fillUnits > 0 {
				// partial fill - use block character
				// fillUnits is how many 1/8ths are filled from the bottom of this row
				localStart := overlapStart - rowStartUnit
				if localStart == 0 {
					// filling from bottom of row
					rowChar = verticalBlocks[fillUnits]
				} else {
					// there's a gap at the bottom, use full block
					// (this handles stacking where lower segment filled some)
					rowChar = '█'
				}
			}
			rowColor = r.color
		}

		// build the row string
		for range b.width {
			rowContent.WriteRune(rowChar)
		}

		style := lipgloss.NewStyle()
		if rowColor != nil {
			style = style.Foreground(rowColor)
		}
		if b.bgColor != nil && rowChar != '█' {
			style = style.Background(b.bgColor)
		}

		lines[row] = style.Render(rowContent.String())
	}

	return strings.Join(lines, "\n")
}

// HorizontalBar represents a horizontal bar (for progress-style displays).
type HorizontalBar struct {
	width    int
	segments []Segment
	bgColor  color.Color
}

// HorizontalBarOption configures a HorizontalBar.
type HorizontalBarOption func(*HorizontalBar)

// WithHorizontalBarBgColor sets the background color.
func WithHorizontalBarBgColor(c color.Color) HorizontalBarOption {
	return func(b *HorizontalBar) {
		b.bgColor = c
	}
}

// NewHorizontalBar creates a new horizontal bar.
func NewHorizontalBar(width int, segments []Segment, opts ...HorizontalBarOption) *HorizontalBar {
	b := &HorizontalBar{
		width:    width,
		segments: segments,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// horizontal block characters for sub-cell resolution.
var horizontalBlocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// render renders the horizontal bar.
func (b *HorizontalBar) Render() string {
	if b.width <= 0 {
		return ""
	}

	var (
		totalUnits = b.width * 8
		result     strings.Builder
		currentPos = 0
	)
	for _, seg := range b.segments {
		units := int(seg.Value * float64(totalUnits))
		if units <= 0 {
			continue
		}

		fullChars := units / 8
		partialUnits := units % 8

		style := lipgloss.NewStyle().Foreground(seg.Color)

		// full characters
		for range fullChars {
			result.WriteString(style.Render("█"))
			currentPos++
		}

		// partial character
		if partialUnits > 0 && currentPos < b.width {
			result.WriteString(style.Render(string(horizontalBlocks[partialUnits])))
			currentPos++
		}
	}

	// fill remaining with background
	if b.bgColor != nil && currentPos < b.width {
		bgStyle := lipgloss.NewStyle().Foreground(b.bgColor)
		for currentPos < b.width {
			result.WriteString(bgStyle.Render("█"))
			currentPos++
		}
	} else {
		for currentPos < b.width {
			result.WriteRune(' ')
			currentPos++
		}
	}

	return result.String()
}
