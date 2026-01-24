package progressbar

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

type ProgressBar struct {
	Width      int
	Percentage float64
	FillColor  color.Color
	EmptyColor color.Color
}

type Option func(*ProgressBar)

func WithFillColor(c color.Color) Option {
	return func(p *ProgressBar) {
		p.FillColor = c
	}
}

func WithEmptyColor(c color.Color) Option {
	return func(p *ProgressBar) {
		p.EmptyColor = c
	}
}

func New(width int, percentage float64, opts ...Option) ProgressBar {
	p := ProgressBar{
		Width:      width,
		Percentage: percentage,
		FillColor:  theme.ColorTeal,
		EmptyColor: theme.ColorDim,
	}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}

func (p ProgressBar) Render() string {
	if p.Width <= 0 {
		return ""
	}

	percentage := p.Percentage
	if percentage > 1 {
		percentage = 1
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(float64(p.Width) * percentage)
	empty := p.Width - filled

	fillStyle := lipgloss.NewStyle().Foreground(p.FillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(p.EmptyColor)

	bar := fillStyle.Render(strings.Repeat("━", filled)) +
		emptyStyle.Render(strings.Repeat("━", empty))

	return bar
}
