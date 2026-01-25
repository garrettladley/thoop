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
	for i, label := range a.labels {
		splitLabels[i] = strings.Split(label, "\n")
		if len(splitLabels[i]) > maxLabelLines {
			maxLabelLines = len(splitLabels[i])
		}
	}

	// calculate positions
	// each label is centered at position: (i + 0.5) * width / len(labels)
	positions := make([]int, len(a.labels))
	for i := range a.labels {
		positions[i] = int((float64(i) + 0.5) * float64(a.width) / float64(len(a.labels)))
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

			startPos := pos - len(text)/2
			if startPos < 0 {
				startPos = 0
			}
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
