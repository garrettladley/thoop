package splash

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

const Duration = 1500 * time.Millisecond

const logo = `
 ▄▄▄▄▄▄▄▄  ▄▄    ▄▄    ▄▄▄▄      ▄▄▄▄    ▄▄▄▄▄▄
 ▀▀▀██▀▀▀  ██    ██   ██▀▀██    ██▀▀██   ██▀▀▀▀█▄
    ██     ██    ██  ██    ██  ██    ██  ██    ██
    ██     ████████  ██    ██  ██    ██  ██████▀
    ██     ██    ██  ██    ██  ██    ██  ██
    ██     ██    ██   ██▄▄██    ██▄▄██   ██
    ▀▀     ▀▀    ▀▀    ▀▀▀▀      ▀▀▀▀    ▀▀`

type TickMsg struct{}

type State struct{}

func LogoView(t theme.Theme) string {
	return t.TextAccent().Render(logo)
}

func View(t theme.Theme, width, height int) string {
	logo := LogoView(t)
	hintStyle := lipgloss.NewStyle().Foreground(theme.ColorDim)
	hint := hintStyle.Render("press any key to continue...")
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		"",
		hint,
	)
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
