package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

// OnboardingPhase represents the current phase of the onboarding flow
type OnboardingPhase uint

const (
	OnboardingPhaseWelcome OnboardingPhase = iota
	OnboardingPhaseAuthenticating
	OnboardingPhaseError
)

type OnboardingState struct {
	Phase    OnboardingPhase
	ErrorMsg string
}

func (m *Model) OnboardingView() string {
	var content string

	switch m.state.onboarding.Phase {
	case OnboardingPhaseWelcome:
		content = m.onboardingWelcomeView()
	case OnboardingPhaseAuthenticating:
		content = m.onboardingAuthenticatingView()
	case OnboardingPhaseError:
		content = m.onboardingErrorView()
	}

	return content
}

func (m *Model) onboardingWelcomeView() string {
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

	logo := m.LogoView()

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

func (m *Model) onboardingAuthenticatingView() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorTeal).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	logo := m.LogoView()

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

func (m *Model) onboardingErrorView() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorLowRecovery).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite)

	errorStyle := lipgloss.NewStyle().
		Foreground(theme.ColorLowRecovery)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	logo := m.LogoView()

	title := titleStyle.Render("Authentication Failed")
	subtitle := subtitleStyle.Render("Something went wrong")
	errorMsg := errorStyle.Render(m.state.onboarding.ErrorMsg)
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
		errorMsg,
		"",
		hint,
	)

	return content
}
