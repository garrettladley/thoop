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
	Label       string
	Value       string
	SubValue    string // e.g., 30-day average shown below main value
	Unit        string
	Width       int
	ProgressBar *progressbar.ProgressBar
	Direction   Direction
	ValueColor  color.Color
}

type Option func(*MetricRow)

func WithProgressBar(percentage float64, fillColor color.Color) Option {
	return func(m *MetricRow) {
		pb := progressbar.New(m.Width, percentage, progressbar.WithFillColor(fillColor))
		m.ProgressBar = &pb
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
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (m MetricRow) Render() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

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
	case percentage >= 85:
		return theme.ColorTeal // optimal
	case percentage >= 70:
		return theme.ColorNeutral // sufficient
	default:
		return theme.ColorOrange // poor
	}
}
