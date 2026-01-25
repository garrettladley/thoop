package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// LegendItem represents a single legend entry.
type LegendItem struct {
	Label string
	Color color.Color
}

// Legend renders a legend with colored indicators.
type Legend struct {
	items      []LegendItem
	horizontal bool
	textColor  color.Color
	spacing    int // space between items in horizontal mode
}

// LegendOption configures a Legend.
type LegendOption func(*Legend)

// WithLegendHorizontal sets horizontal layout.
func WithLegendHorizontal(h bool) LegendOption {
	return func(l *Legend) {
		l.horizontal = h
	}
}

// WithLegendTextColor sets the label text color.
func WithLegendTextColor(c color.Color) LegendOption {
	return func(l *Legend) {
		l.textColor = c
	}
}

// WithLegendSpacing sets the spacing between items.
func WithLegendSpacing(s int) LegendOption {
	return func(l *Legend) {
		l.spacing = s
	}
}

// NewLegend creates a new legend.
func NewLegend(items []LegendItem, opts ...LegendOption) *Legend {
	l := &Legend{
		items:      items,
		horizontal: true,
		textColor:  theme.ColorWhite,
		spacing:    2,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Render renders the legend.
func (l *Legend) Render() string {
	if len(l.items) == 0 {
		return ""
	}

	textStyle := lipgloss.NewStyle().Foreground(l.textColor)

	parts := make([]string, 0, len(l.items))
	for _, item := range l.items {
		boxStyle := lipgloss.NewStyle().Foreground(item.Color)
		entry := boxStyle.Render(theme.SymbolSquareFilled) + " " + textStyle.Render(item.Label)
		parts = append(parts, entry)
	}

	if l.horizontal {
		spacing := strings.Repeat(" ", l.spacing)
		return strings.Join(parts, spacing)
	}

	return strings.Join(parts, "\n")
}
