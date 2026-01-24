package tui

import (
	"errors"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/tui/components/footer"
	"github.com/garrettladley/thoop/internal/tui/page"
	"github.com/garrettladley/thoop/internal/tui/page/dashboard"
	"github.com/garrettladley/thoop/internal/tui/page/onboarding"
	"github.com/garrettladley/thoop/internal/tui/page/splash"
	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/xslog"
)

var _ tea.Model = (*Model)(nil)

const (
	tokenCheckInterval    = 5 * time.Minute
	tokenRefreshThreshold = 15 * time.Minute
	splashMinDuration     = 1 * time.Second
)

type splashMinElapsedMsg struct{}

type state struct {
	splash          splash.State
	onboarding      onboarding.State
	dashboard       dashboard.State
	authChecked     bool
	splashMinPassed bool
}

type Model struct {
	ready          bool
	page           page.ID
	viewportWidth  int
	viewportHeight int
	theme          theme.Theme
	state          state
	deps           Deps
	sseOnce        sync.Once
}

func New(deps Deps) Model {
	return Model{
		page:  page.Splash,
		theme: theme.New(),
		deps:  deps,
		state: state{
			splash:     splash.State{},
			onboarding: onboarding.State{},
			dashboard:  dashboard.State{},
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(splash.Duration, func(t time.Time) tea.Msg {
			return splash.TickMsg{}
		}),
		tea.Tick(splashMinDuration, func(t time.Time) tea.Msg {
			return splashMinElapsedMsg{}
		}),
		onboarding.CheckAuthCmd(m.deps.Ctx, m.deps.TokenChecker),
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

	case splash.TickMsg:
		return m.handleSplashTick()

	case splashMinElapsedMsg:
		m.state.splashMinPassed = true
		return m, m.maybeTransitionToDashboard()

	case onboarding.AuthStatusMsg:
		return m.handleAuthStatus(msg)

	case onboarding.AuthFlowResultMsg:
		return m.handleAuthFlowResult(msg)

	case onboarding.TokenCheckTickMsg:
		return m.handleTokenCheckTick()

	case onboarding.TokenRefreshResultMsg:
		return m.handleTokenRefreshResult(msg)

	case dashboard.CycleMsg:
		if msg.Err != nil {
			return m, nil
		}
		if msg.Cycle != nil {
			m.state.dashboard.CycleID = msg.Cycle.ID
			m.state.dashboard.CurrentCycle = msg.Cycle
			if msg.Cycle.Score != nil {
				m.state.dashboard.StrainScore = &msg.Cycle.Score.Strain
			}
			m.state.dashboard.SetPending()
			return m, tea.Batch(
				dashboard.FetchSleepCmd(m.deps.Ctx, m.deps.WhoopClient, msg.Cycle.ID),
				dashboard.FetchRecoveryCmd(m.deps.Ctx, m.deps.WhoopClient, msg.Cycle.ID),
			)
		}
		return m, nil

	case dashboard.SleepMsg:
		if msg.Err == nil && msg.Sleep != nil {
			m.state.dashboard.CurrentSleep = msg.Sleep
			if msg.Sleep.Score != nil {
				m.state.dashboard.SleepScore = &msg.Sleep.Score.SleepPerformancePercentage
			}
		}
		m.state.dashboard.ClearPendingSleep()
		return m, m.maybeTransitionToDashboard()

	case dashboard.RecoveryMsg:
		if msg.Err == nil && msg.Recovery != nil {
			m.state.dashboard.CurrentRecovery = msg.Recovery
			if msg.Recovery.Score != nil {
				m.state.dashboard.RecoveryScore = &msg.Recovery.Score.RecoveryScore
			}
		}
		m.state.dashboard.ClearPendingRecovery()
		return m, m.maybeTransitionToDashboard()

	case dashboard.HistoricalDataMsg:
		if msg.Err != nil {
			m.deps.Logger.ErrorContext(m.deps.Ctx, "historical data fetch failed",
				xslog.Source(msg.ErrSource),
				xslog.Error(msg.Err))
		} else {
			m.state.dashboard.Averages = dashboard.ComputeAverages(msg.Recoveries, msg.Cycles, msg.Sleeps)
		}
		return m, nil

	case NotificationMsg:
		if m.page == page.Dashboard {
			return m, tea.Batch(
				dashboard.FetchCycleCmd(m.deps.Ctx, m.deps.WhoopClient),
				ListenNotificationsCmd(m.deps.Ctx, m.deps.NotificationChan, m.deps.NotifProcessor, m.deps.SSEClient),
			)
		}
		return m, ListenNotificationsCmd(m.deps.Ctx, m.deps.NotificationChan, m.deps.NotifProcessor, m.deps.SSEClient)

	case SSEDisconnectedMsg:
		if msg.Err != nil {
			m.deps.Logger.WarnContext(m.deps.Ctx, "SSE disconnected", xslog.Error(msg.Err))
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
		case page.Onboarding:
			switch m.state.onboarding.Phase {
			case onboarding.PhaseWelcome, onboarding.PhaseError:
				m.state.onboarding.Phase = onboarding.PhaseAuthenticating
				m.state.onboarding.ErrorMsg = ""
				return m, onboarding.StartAuthFlowCmd(m.deps.Ctx, m.deps.AuthFlow)
			default:
			}
		case page.Dashboard:
			if m.state.dashboard.ActiveTab == dashboard.TabOverview {
				m.state.dashboard.ActiveTab = dashboard.TabRecovery
				return m, nil
			}
		default:
		}
	case "right", "l":
		if m.page == page.Dashboard {
			m.state.dashboard.ActiveTab = dashboard.NextTab(m.state.dashboard.ActiveTab)
			return m, nil
		}
	case "left", "h":
		if m.page == page.Dashboard {
			m.state.dashboard.ActiveTab = dashboard.PrevTab(m.state.dashboard.ActiveTab)
			return m, nil
		}
	case "j", "k", "up", "down":
		if m.page == page.Dashboard && m.state.dashboard.ActiveTab == dashboard.TabOverview {
			m.state.dashboard.ActiveTab = dashboard.TabRecovery
			return m, nil
		}
	case "esc":
		if m.page == page.Dashboard && m.state.dashboard.ActiveTab != dashboard.TabOverview {
			m.state.dashboard.ActiveTab = dashboard.TabOverview
			return m, nil
		}
	default:
		// skip splash on any keypress (only if auth is checked)
		if m.page == page.Splash && m.state.authChecked {
			if m.state.dashboard.AuthIndicator.Authenticated {
				m.page = page.Dashboard
				return m, m.startDashboardServices()
			} else {
				m.page = page.Onboarding
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
			return splash.TickMsg{}
		})
	}

	if m.state.dashboard.AuthIndicator.Authenticated {
		m.page = page.Dashboard
		return m, m.startDashboardServices()
	}

	m.page = page.Onboarding
	return m, nil
}

func (m *Model) maybeTransitionToDashboard() tea.Cmd {
	// auto-transition from splash when data is ready and min time elapsed
	if m.page == page.Splash && m.state.splashMinPassed && m.state.dashboard.DataReady() {
		m.page = page.Dashboard
		return m.startDashboardServices()
	}
	return nil
}

func (m *Model) handleAuthStatus(msg onboarding.AuthStatusMsg) (tea.Model, tea.Cmd) {
	m.state.authChecked = true
	m.state.dashboard.AuthIndicator.Checked = true

	if msg.Err == nil {
		m.state.dashboard.AuthIndicator.Authenticated = msg.HasToken
	}

	// start fetching data during splash if authenticated
	if m.state.dashboard.AuthIndicator.Authenticated {
		return m, tea.Batch(
			dashboard.FetchCycleCmd(m.deps.Ctx, m.deps.WhoopClient),
			dashboard.FetchHistoricalDataCmd(m.deps.Ctx, m.deps.WhoopClient),
		)
	}

	return m, nil
}

func (m *Model) handleAuthFlowResult(msg onboarding.AuthFlowResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.state.onboarding.Phase = onboarding.PhaseError
		m.state.onboarding.ErrorMsg = msg.Err.Error()
		return m, nil
	}

	if msg.APIKey != "" {
		m.deps.WhoopClient.SetAPIKey(msg.APIKey)
		m.deps.SSEClient.SetAPIKey(msg.APIKey)
	}

	m.state.dashboard.AuthIndicator.Authenticated = true
	m.page = page.Dashboard
	return m, tea.Batch(
		dashboard.FetchCycleCmd(m.deps.Ctx, m.deps.WhoopClient),
		dashboard.FetchHistoricalDataCmd(m.deps.Ctx, m.deps.WhoopClient),
		m.startDashboardServices(),
	)
}

func (m *Model) handleTokenCheckTick() (tea.Model, tea.Cmd) {
	if m.page != page.Dashboard {
		return m, nil
	}

	return m, onboarding.RefreshTokenIfNeededCmd(m.deps.Ctx, m.deps.TokenSource, tokenRefreshThreshold)
}

func (m *Model) handleTokenRefreshResult(msg onboarding.TokenRefreshResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		if errors.Is(msg.Err, oauth.ErrNoToken) || errors.Is(msg.Err, oauth.ErrTokenExpired) {
			m.state.dashboard.AuthIndicator.Authenticated = false
			m.page = page.Onboarding
			m.state.onboarding.Phase = onboarding.PhaseWelcome
			return m, nil
		}
	}

	return m, onboarding.TokenCheckTickCmd(tokenCheckInterval)
}

func (m *Model) startDashboardServices() tea.Cmd {
	cmds := []tea.Cmd{
		onboarding.TokenCheckTickCmd(tokenCheckInterval),
	}

	m.sseOnce.Do(func() {
		cmds = append(cmds,
			StartSSECmd(m.deps.Ctx, m.deps.SSEClient, m.deps.NotificationChan),
			ListenNotificationsCmd(m.deps.Ctx, m.deps.NotificationChan, m.deps.NotifProcessor, m.deps.SSEClient),
		)
	})

	return tea.Batch(cmds...)
}

func (m *Model) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true

	// splash and onboarding use pure black BG, dashboard uses default dark
	switch m.page {
	case page.Splash, page.Onboarding:
		view.BackgroundColor = theme.ColorBlack
	default:
		view.BackgroundColor = m.theme.Background()
	}

	if !m.ready {
		return view
	}

	var content string
	switch m.page {
	case page.Splash:
		content = splash.View(m.theme, m.viewportWidth, m.viewportHeight)
	case page.Onboarding:
		content = onboarding.View(m.theme, m.state.onboarding, m.viewportWidth, m.viewportHeight)
	case page.Dashboard:
		gauges := dashboard.View(m.state.dashboard, m.viewportWidth, m.viewportHeight)

		f := footer.New(dashboard.AuthIndicatorView(m.state.dashboard), m.viewportWidth)

		switch m.state.dashboard.ActiveTab {
		case dashboard.TabOverview:
			f = f.WithNavHints("← sleep    ↑↓ recovery    → strain")
		default:
			f = f.WithNavHints("esc back    ←/→ navigate")
		}

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
