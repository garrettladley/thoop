package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	background color.Color
	foreground color.Color
	base       lipgloss.Style
}

func New() Theme {
	var t Theme

	t.background = ColorBgDark
	t.foreground = ColorWhite
	t.base = lipgloss.NewStyle().Foreground(t.foreground)

	return t
}

func (t Theme) Base() lipgloss.Style {
	return t.base
}

func (t Theme) TextAccent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.foreground)
}

func (t Theme) Background() color.Color {
	return t.background
}

func (t Theme) Foreground() color.Color {
	return t.foreground
}
