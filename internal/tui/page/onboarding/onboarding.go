package onboarding

import (
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/page/splash"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

type Phase uint

const (
	PhaseWelcome Phase = iota
	PhaseAuthenticating
	PhaseError
)

type State struct {
	Phase    Phase
	ErrorMsg string
}

func View(t theme.Theme, state State, width, height int) string {
	var content string

	switch state.Phase {
	case PhaseWelcome:
		content = welcomeView(t)
	case PhaseAuthenticating:
		content = authenticatingView(t)
	case PhaseError:
		content = errorView(t, state.ErrorMsg)
	}

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func welcomeView(t theme.Theme) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorTeal).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	buttonStyle := lipgloss.NewStyle().
		Foreground(theme.ColorBgDark).
		Background(theme.ColorTeal).
		Padding(0, 2).
		Bold(true)

	logo := splash.LogoView(t)

	title := titleStyle.Render("Connect to WHOOP")
	subtitle := subtitleStyle.Render("Authenticate to view your health data")
	button := buttonStyle.Render("Press Enter to authenticate")
	hint := hintStyle.Render("This will open your browser")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		"",
		title,
		"",
		subtitle,
		"",
		"",
		button,
		"",
		hint,
	)

	return content
}

func authenticatingView(t theme.Theme) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorTeal).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	logo := splash.LogoView(t)

	title := titleStyle.Render("Authenticating...")
	subtitle := subtitleStyle.Render("Complete the login in your browser")
	hint := hintStyle.Render("Waiting for authorization...")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		"",
		title,
		"",
		subtitle,
		"",
		"",
		hint,
	)

	return content
}

func errorView(t theme.Theme, errorMsg string) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorLowRecovery).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite)

	errorStyle := lipgloss.NewStyle().
		Foreground(theme.ColorLowRecovery)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	logo := splash.LogoView(t)

	title := titleStyle.Render("Authentication Failed")
	subtitle := subtitleStyle.Render("Something went wrong")
	errorText := errorStyle.Render(errorMsg)
	hint := hintStyle.Render("Press Enter to try again, or q to quit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		"",
		title,
		"",
		subtitle,
		"",
		errorText,
		"",
		hint,
	)

	return content
}
