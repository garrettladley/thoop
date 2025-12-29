package tui

import (
	"errors"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/tui/commands"
	"github.com/garrettladley/thoop/internal/tui/components/footer"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

var _ tea.Model = (*Model)(nil)

type page uint

const (
	splashPage page = iota
	onboardingPage
	dashboardPage
)

const (
	tokenCheckInterval    = 5 * time.Minute
	tokenRefreshThreshold = 15 * time.Minute
)

type state struct {
	splash      SplashState
	onboarding  OnboardingState
	dashboard   DashboardState
	authChecked bool
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
			splash:     SplashState{},
			onboarding: OnboardingState{},
			dashboard:  DashboardState{},
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(splashDuration, func(t time.Time) tea.Msg {
			return SplashTickMsg{}
		}),
		commands.CheckAuthCmd(m.deps.Ctx, m.deps.TokenChecker),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewportWidth = msg.Width
		m.viewportHeight = msg.Height
		m.ready = true

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case SplashTickMsg:
		return m.handleSplashTick()

	case commands.AuthStatusMsg:
		return m.handleAuthStatus(msg)

	case commands.AuthFlowResultMsg:
		return m.handleAuthFlowResult(msg)

	case commands.TokenCheckTickMsg:
		return m.handleTokenCheckTick()

	case commands.TokenRefreshResultMsg:
		return m.handleTokenRefreshResult(msg)

	case commands.CycleMsg:
		if msg.Err != nil {
			return m, nil
		}
		if msg.Cycle != nil {
			m.state.dashboard.CycleID = msg.Cycle.ID
			if msg.Cycle.Score != nil {
				m.state.dashboard.StrainScore = &msg.Cycle.Score.Strain
			}
			return m, tea.Batch(
				commands.FetchSleepCmd(m.deps.Ctx, m.deps.WhoopClient, msg.Cycle.ID),
				commands.FetchRecoveryCmd(m.deps.Ctx, m.deps.WhoopClient, msg.Cycle.ID),
			)
		}
		return m, nil

	case commands.SleepMsg:
		if msg.Err == nil && msg.Sleep != nil && msg.Sleep.Score != nil {
			m.state.dashboard.SleepScore = &msg.Sleep.Score.SleepPerformancePercentage
		}
		return m, nil

	case commands.RecoveryMsg:
		if msg.Err == nil && msg.Recovery != nil && msg.Recovery.Score != nil {
			m.state.dashboard.RecoveryScore = &msg.Recovery.Score.RecoveryScore
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.deps.Cancel()
		return m, tea.Quit
	case "enter":
		switch m.page {
		case onboardingPage:
			switch m.state.onboarding.Phase {
			case OnboardingPhaseWelcome, OnboardingPhaseError:
				m.state.onboarding.Phase = OnboardingPhaseAuthenticating
				m.state.onboarding.ErrorMsg = ""
				return m, commands.StartAuthFlowCmd(m.deps.Ctx, m.deps.AuthFlow)
			}
		}
	default:
		// skip splash on any keypress (only if auth is checked)
		if m.page == splashPage && m.state.authChecked {
			if m.state.dashboard.AuthIndicator.Authenticated {
				m.page = dashboardPage
				return m, m.startDashboard()
			} else {
				m.page = onboardingPage
			}
		}
	}
	return m, nil
}

func (m *Model) handleSplashTick() (tea.Model, tea.Cmd) {
	// only transition if auth status is known
	if !m.state.authChecked {
		// auth check still pending, wait a bit more
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return SplashTickMsg{}
		})
	}

	if m.state.dashboard.AuthIndicator.Authenticated {
		m.page = dashboardPage
		return m, m.startDashboard()
	}

	m.page = onboardingPage
	return m, nil
}

func (m *Model) handleAuthStatus(msg commands.AuthStatusMsg) (tea.Model, tea.Cmd) {
	m.state.authChecked = true
	m.state.dashboard.AuthIndicator.Checked = true

	if msg.Err == nil {
		m.state.dashboard.AuthIndicator.Authenticated = msg.HasToken
	}

	return m, nil
}

func (m *Model) handleAuthFlowResult(msg commands.AuthFlowResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.state.onboarding.Phase = OnboardingPhaseError
		m.state.onboarding.ErrorMsg = msg.Err.Error()
		return m, nil
	}

	m.state.dashboard.AuthIndicator.Authenticated = true
	m.page = dashboardPage
	return m, m.startDashboard()
}

func (m *Model) handleTokenCheckTick() (tea.Model, tea.Cmd) {
	if m.page != dashboardPage {
		return m, nil
	}

	return m, commands.RefreshTokenIfNeededCmd(m.deps.Ctx, m.deps.TokenSource, tokenRefreshThreshold)
}

func (m *Model) handleTokenRefreshResult(msg commands.TokenRefreshResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		if errors.Is(msg.Err, oauth.ErrNoToken) || errors.Is(msg.Err, oauth.ErrTokenExpired) {
			m.state.dashboard.AuthIndicator.Authenticated = false
			m.page = onboardingPage
			m.state.onboarding.Phase = OnboardingPhaseWelcome
			return m, nil
		}
	}

	return m, commands.TokenCheckTickCmd(tokenCheckInterval)
}

func (m *Model) startDashboard() tea.Cmd {
	return tea.Batch(
		commands.FetchCycleCmd(m.deps.Ctx, m.deps.WhoopClient),
		commands.TokenCheckTickCmd(tokenCheckInterval),
	)
}

func (m *Model) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true

	// splash and onboarding use pure black BG, dashboard uses default dark
	if m.page == splashPage || m.page == onboardingPage {
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
	case onboardingPage:
		content = m.OnboardingView()
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

		f := footer.New(m.AuthIndicatorView(), m.viewportWidth)

		footerOverlay := lipgloss.Place(
			m.viewportWidth,
			m.viewportHeight,
			lipgloss.Left,
			lipgloss.Bottom,
			f.Render(),
		)

		content = m.overlayStrings(gauges, footerOverlay)
	}

	view.SetContent(content)
	return view
}

func (m *Model) overlayStrings(base, overlay string) string {
	var (
		baseLines    = strings.Split(base, "\n")
		overlayLines = strings.Split(overlay, "\n")
	)

	maxLines := max(len(overlayLines), len(baseLines))

	result := make([]string, maxLines)
	for i := range maxLines {
		var (
			baseLine    string
			overlayLine string
		)
		if i < len(baseLines) {
			baseLine = baseLines[i]
		}
		if i < len(overlayLines) {
			overlayLine = overlayLines[i]
		}

		var (
			baseRunes    = []rune(baseLine)
			overlayRunes = []rune(overlayLine)
		)

		maxLen := max(len(overlayRunes), len(baseRunes))

		merged := make([]rune, maxLen)
		for j := range maxLen {
			var (
				baseChar    = ' '
				overlayChar = ' '
			)
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
