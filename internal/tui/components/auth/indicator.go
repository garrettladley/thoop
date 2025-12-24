package auth

import (
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

const statusDot = "●"

type Indicator struct {
	Checked       bool
	Authenticated bool
}

func (a Indicator) Render() string {
	if !a.Checked {
		return lipgloss.NewStyle().
			Foreground(theme.ColorBgLight).
			Render(statusDot + " checking...")
	}

	if a.Authenticated {
		return lipgloss.NewStyle().
			Foreground(theme.ColorHighRecovery).
			Render(statusDot + " connected")
	}

	return lipgloss.NewStyle().
		Foreground(theme.ColorLowRecovery).
		Render(statusDot + " not connected")
}
