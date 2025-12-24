package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

var _ tea.Model = (*Model)(nil)

type page uint

const (
	splashPage page = iota
	dashboardPage
)

type state struct {
	splash    SplashState
	dashboard DashboardState
}

type Model struct {
	ready          bool
	page           page
	viewportWidth  int
	viewportHeight int
	theme          theme.Theme
	state          state
	deps           Deps
}

func New(deps Deps) Model {
	return Model{
		page:  splashPage,
		theme: theme.New(),
		deps:  deps,
		state: state{
			splash:    SplashState{},
			dashboard: DashboardState{},
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(splashDuration, func(t time.Time) tea.Msg {
			return SplashTickMsg{}
		}),
		checkAuthCmd(m.deps.TokenChecker),
		fetchMetricsCmd(m.deps.WhoopClient),
	)
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

	// splash timer expired - transition to dashboard
	case SplashTickMsg:
		m.page = dashboardPage

	case AuthStatusMsg:
		m.state.dashboard.AuthIndicator.Checked = true
		if msg.Err == nil {
			m.state.dashboard.AuthIndicator.Authenticated = msg.HasToken
		}

	case MetricsDataMsg:
		if msg.Err == nil {
			m.state.dashboard.SleepScore = msg.Sleep
			m.state.dashboard.RecoveryScore = msg.Recovery
			m.state.dashboard.StrainScore = msg.Strain
		}
	}

	return m, nil
}

func (m *Model) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true

	// splash uses pure black BG, everything else uses default dark
	if m.page == splashPage {
		view.BackgroundColor = theme.ColorBlack
	} else {
		view.BackgroundColor = m.theme.Background()
	}

	if !m.ready {
		return view
	}

	var content string
	switch m.page {
	case splashPage:
		content = m.SplashView()
		content = lipgloss.Place(
			m.viewportWidth,
			m.viewportHeight,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)
	case dashboardPage:
		gauges := lipgloss.Place(
			m.viewportWidth,
			m.viewportHeight,
			lipgloss.Center,
			lipgloss.Center,
			m.DashboardView(),
		)

		// place auth indicator at absolute bottom right
		authIndicator := lipgloss.NewStyle().
			PaddingRight(2).
			PaddingBottom(1).
			Render(m.AuthIndicatorView())

		authOverlay := lipgloss.Place(
			m.viewportWidth,
			m.viewportHeight,
			lipgloss.Right,
			lipgloss.Bottom,
			authIndicator,
		)

		content = m.overlayStrings(gauges, authOverlay)
	}

	view.SetContent(content)
	return view
}

func (m *Model) overlayStrings(base, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	maxLines := len(baseLines)
	if len(overlayLines) > maxLines {
		maxLines = len(overlayLines)
	}

	result := make([]string, maxLines)
	for i := range maxLines {
		var baseLine, overlayLine string
		if i < len(baseLines) {
			baseLine = baseLines[i]
		}
		if i < len(overlayLines) {
			overlayLine = overlayLines[i]
		}

		baseRunes := []rune(baseLine)
		overlayRunes := []rune(overlayLine)

		maxLen := len(baseRunes)
		if len(overlayRunes) > maxLen {
			maxLen = len(overlayRunes)
		}

		merged := make([]rune, maxLen)
		for j := 0; j < maxLen; j++ {
			baseChar, overlayChar := ' ', ' '
			if j < len(baseRunes) {
				baseChar = baseRunes[j]
			}
			if j < len(overlayRunes) {
				overlayChar = overlayRunes[j]
			}

			if overlayChar != ' ' {
				merged[j] = overlayChar
			} else {
				merged[j] = baseChar
			}
		}
		result[i] = string(merged)
	}

	return strings.Join(result, "\n")
}
