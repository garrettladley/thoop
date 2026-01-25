package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// Axis renders an x-axis with labels.
type Axis struct {
	labels    []string
	width     int
	showLine  bool
	labelGap  int // minimum gap between labels
	textColor color.Color
	positions []int // explicit label positions (centers)
}

// AxisOption configures an Axis.
type AxisOption func(*Axis)

// WithAxisLine shows a line separator above the labels.
func WithAxisLine(show bool) AxisOption {
	return func(a *Axis) {
		a.showLine = show
	}
}

// WithAxisLabelGap sets the minimum gap between labels.
func WithAxisLabelGap(gap int) AxisOption {
	return func(a *Axis) {
		a.labelGap = gap
	}
}

// WithAxisTextColor sets the label text color.
func WithAxisTextColor(c color.Color) AxisOption {
	return func(a *Axis) {
		a.textColor = c
	}
}

// WithAxisPositions sets explicit center positions for labels.
func WithAxisPositions(positions []int) AxisOption {
	return func(a *Axis) {
		a.positions = positions
	}
}

// NewAxis creates a new axis with the given labels.
func NewAxis(labels []string, width int, opts ...AxisOption) *Axis {
	a := &Axis{
		labels:    labels,
		width:     width,
		labelGap:  1,
		textColor: theme.ColorDim,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Render renders the axis.
func (a *Axis) Render() string {
	if len(a.labels) == 0 || a.width <= 0 {
		return ""
	}

	var lines []string

	if a.showLine {
		lines = append(lines, strings.Repeat("─", a.width))
	}

	// calculate label positions
	// labels are evenly distributed across the width
	labelStyle := lipgloss.NewStyle().Foreground(a.textColor)

	// for multi-line labels (e.g., "Mon\n12"), split them
	maxLabelLines := 1
	splitLabels := make([][]string, len(a.labels))
	labelWidths := make([]int, len(a.labels)) // max width per label (for centering all lines)
	for i, label := range a.labels {
		splitLabels[i] = strings.Split(label, "\n")
		if len(splitLabels[i]) > maxLabelLines {
			maxLabelLines = len(splitLabels[i])
		}
		// find max line width for this label
		for _, part := range splitLabels[i] {
			if len(part) > labelWidths[i] {
				labelWidths[i] = len(part)
			}
		}
	}

	// use explicit positions if provided, otherwise calculate to match Line renderer
	positions := a.positions
	if len(positions) != len(a.labels) {
		// match the positioning from the Line renderer:
		// braille canvas is width*2 dots wide
		// xStep = (dotsWidth-1) / (len(points)-1)
		// for each point i, dotX = i * xStep
		// character position = int(dotX / 2)
		dotsWidth := a.width * 2
		var xStep float64
		if len(a.labels) == 1 {
			xStep = 0
		} else {
			xStep = float64(dotsWidth-1) / float64(len(a.labels)-1)
		}
		positions = make([]int, len(a.labels))
		for i := range a.labels {
			dotX := int(float64(i) * xStep)
			positions[i] = dotX / 2
		}
	}

	// render each line of labels
	for lineIdx := range maxLabelLines {
		row := make([]rune, a.width)
		for i := range row {
			row[i] = ' '
		}

		for labelIdx, parts := range splitLabels {
			if lineIdx >= len(parts) {
				continue
			}

			text := parts[lineIdx]
			pos := positions[labelIdx]
			labelWidth := labelWidths[labelIdx]

			// center the label block at pos, then center text within that block
			blockStart := pos - labelWidth/2
			textOffset := (labelWidth - len(text)) / 2
			startPos := blockStart + textOffset

			startPos = max(0, startPos)
			if startPos+len(text) > a.width {
				startPos = a.width - len(text)
			}

			for i, r := range text {
				if startPos+i < a.width {
					row[startPos+i] = r
				}
			}
		}

		lines = append(lines, labelStyle.Render(string(row)))
	}

	return strings.Join(lines, "\n")
}
