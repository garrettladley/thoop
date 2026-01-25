package metric_row

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/components/progressbar"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

const (
	symbolUp      = "▲"
	symbolDown    = "▼"
	symbolNeutral = "●"
)

type Direction int

const (
	DirectionNone     Direction = iota
	DirectionUp                 // Higher is good (teal up)
	DirectionDown               // Lower is bad (orange down)
	DirectionUpBad              // Higher is bad (orange up)
	DirectionDownGood           // Lower is good (teal down)
	DirectionNeutral            // No change (grey circle)
)

type MetricRow struct {
	Label                string
	Value                string
	SubValue             string // e.g., 30-day average shown below main value
	Unit                 string
	Width                int
	ProgressBar          *progressbar.ProgressBar
	SegmentedProgressBar *progressbar.SegmentedProgressBar
	Direction            Direction
	ValueColor           color.Color
	LabelColor           color.Color
}

type Option func(*MetricRow)

func WithProgressBar(percentage float64, fillColor color.Color) Option {
	return func(m *MetricRow) {
		pb := progressbar.New(m.Width, percentage, progressbar.WithFillColor(fillColor))
		m.ProgressBar = &pb
	}
}

func WithSegmentedProgressBar(percentage float64, segmentWidth int) Option {
	return func(m *MetricRow) {
		spb := progressbar.NewSegmented(percentage, progressbar.WithSegmentWidth(segmentWidth))
		m.SegmentedProgressBar = &spb
	}
}

// WithSegmentedProgressBarThresholds creates a segmented progress bar with custom thresholds.
// sufficientThreshold: minimum % for sufficient (yellow)
// optimalThreshold: minimum % for optimal (green)
func WithSegmentedProgressBarThresholds(percentage float64, segmentWidth int, sufficientThreshold, optimalThreshold float64) Option {
	return func(m *MetricRow) {
		spb := progressbar.NewSegmented(percentage,
			progressbar.WithSegmentWidth(segmentWidth),
			progressbar.WithSegmentedThresholds(sufficientThreshold, optimalThreshold),
		)
		m.SegmentedProgressBar = &spb
	}
}

func WithLabelColor(c color.Color) Option {
	return func(m *MetricRow) {
		m.LabelColor = c
	}
}

func WithDirection(d Direction) Option {
	return func(m *MetricRow) {
		m.Direction = d
	}
}

func WithValueColor(c color.Color) Option {
	return func(m *MetricRow) {
		m.ValueColor = c
	}
}

func WithSubValue(subValue string) Option {
	return func(m *MetricRow) {
		m.SubValue = subValue
	}
}

func WithUnit(unit string) Option {
	return func(m *MetricRow) {
		m.Unit = unit
	}
}

func New(label, value string, width int, opts ...Option) MetricRow {
	m := MetricRow{
		Label:      label,
		Value:      value,
		Width:      width,
		Direction:  DirectionNone,
		ValueColor: theme.ColorWhite,
		LabelColor: theme.ColorDim,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (m MetricRow) Render() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(m.LabelColor)

	valueStyle := lipgloss.NewStyle().
		Foreground(m.ValueColor).
		Bold(true)

	subValueStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	directionStr := ""
	switch m.Direction {
	case DirectionNone:
		// no indicator shown
	case DirectionUp:
		directionStr = " " + lipgloss.NewStyle().
			Foreground(theme.ColorTeal).
			Render(symbolUp)
	case DirectionDown:
		directionStr = " " + lipgloss.NewStyle().
			Foreground(theme.ColorOrange).
			Render(symbolDown)
	case DirectionUpBad:
		directionStr = " " + lipgloss.NewStyle().
			Foreground(theme.ColorOrange).
			Render(symbolUp)
	case DirectionDownGood:
		directionStr = " " + lipgloss.NewStyle().
			Foreground(theme.ColorTeal).
			Render(symbolDown)
	case DirectionNeutral:
		directionStr = " " + lipgloss.NewStyle().
			Foreground(theme.ColorNeutral).
			Render(symbolNeutral)
	}

	labelText := labelStyle.Render(m.Label)
	valueText := valueStyle.Render(m.Value+m.Unit) + directionStr

	// for segmented bar, layout is: Label ... [bar] [value] (bar+value right-aligned)
	if m.SegmentedProgressBar != nil {
		barStr := m.SegmentedProgressBar.Render()
		barWidth := lipgloss.Width(barStr)
		labelWidth := lipgloss.Width(labelText)
		valueWidth := lipgloss.Width(valueText)

		// bar and value are grouped together on the right
		rightGroup := barStr + "  " + valueText
		rightWidth := barWidth + 2 + valueWidth

		padding := m.Width - labelWidth - rightWidth
		padding = max(padding, 1)

		return labelText + fmt.Sprintf("%*s", padding, "") + rightGroup
	}

	labelWidth := lipgloss.Width(labelText)
	valueWidth := lipgloss.Width(valueText)
	padding := m.Width - labelWidth - valueWidth

	padding = max(padding, 1)

	row := labelText + fmt.Sprintf("%*s", padding, "") + valueText

	// add sub-value row if present (30-day average below current value)
	if m.SubValue != "" {
		subValueText := subValueStyle.Render(m.SubValue + m.Unit)
		subPadding := m.Width - lipgloss.Width(subValueText)
		subRow := fmt.Sprintf("%*s", subPadding, "") + subValueText
		row = lipgloss.JoinVertical(lipgloss.Left, row, subRow)
	}

	if m.ProgressBar != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			row,
			m.ProgressBar.Render(),
		)
	}

	return row
}

func PercentageColor(percentage float64) color.Color {
	switch {
	case percentage > 85:
		return theme.ColorTeal // optimal
	case percentage >= 70:
		return theme.ColorNeutral // sufficient
	default:
		return theme.ColorOrange // poor
	}
}
