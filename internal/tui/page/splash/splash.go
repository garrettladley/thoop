package splash

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

const Duration = 1500 * time.Millisecond

const Logo = `
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
	return t.TextAccent().Render(Logo)
}

func View(t theme.Theme, width, height int) string {
	logo := LogoView(t)
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		logo,
	)
}
