package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var _ tea.Model = (*Model)(nil)

type page int

const (
	splashPage page = iota
)

type state struct {
	splash SplashState
}

type Model struct {
	ready          bool
	page           page
	viewportWidth  int
	viewportHeight int
	theme          Theme
	state          state
}

func New() Model {
	return Model{
		page:  splashPage,
		theme: NewTheme(),
		state: state{
			splash: SplashState{},
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewportWidth = msg.Width
		m.viewportHeight = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *Model) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true
	view.BackgroundColor = m.theme.background

	if !m.ready {
		return view
	}

	content := m.SplashView()
	styled := lipgloss.NewStyle().
		Width(m.viewportWidth).
		Height(m.viewportHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)

	view.SetContent(styled)
	return view
}
