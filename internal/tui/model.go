package tui

import (
	"errors"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/cache"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/tui/components/calendar"
	"github.com/garrettladley/thoop/internal/tui/components/chart"
	"github.com/garrettladley/thoop/internal/tui/components/footer"
	"github.com/garrettladley/thoop/internal/tui/page"
	"github.com/garrettladley/thoop/internal/tui/page/dashboard"
	"github.com/garrettladley/thoop/internal/tui/page/onboarding"
	"github.com/garrettladley/thoop/internal/tui/page/splash"
	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/xslices"
	"github.com/garrettladley/thoop/internal/xslog"
	"github.com/garrettladley/thoop/internal/xtime"
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

	case tea.MouseWheelMsg:
		if m.page == page.Dashboard && m.state.dashboard.ActiveTab != dashboard.TabOverview {
			switch msg.Button {
			case tea.MouseWheelUp:
				m.state.dashboard.ScrollOffset -= 3
				if m.state.dashboard.ScrollOffset < 0 {
					m.state.dashboard.ScrollOffset = 0
				}
			case tea.MouseWheelDown:
				m.state.dashboard.ScrollOffset += 3
				// cap at reasonable max to prevent scrolling into void
				maxScroll := m.viewportHeight * 2
				if m.state.dashboard.ScrollOffset > maxScroll {
					m.state.dashboard.ScrollOffset = maxScroll
				}
			}
			return m, nil
		}

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
			refDate := m.state.dashboard.EffectiveDate()
			return m, tea.Batch(
				dashboard.FetchSleepCmd(m.deps.Ctx, m.deps.CacheService, msg.Cycle.ID),
				dashboard.FetchRecoveryCmd(m.deps.Ctx, m.deps.CacheService, msg.Cycle.ID),
				dashboard.FetchWorkoutsForDateCmd(m.deps.Ctx, m.deps.CacheService, msg.Cycle.Start, refDate),
			)
		}
		return m, nil

	case dashboard.WorkoutsMsg:
		if msg.Err == nil {
			m.state.dashboard.TodaysWorkouts = msg.Workouts
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
		// Clear chart cache when new data arrives
		chart.ClearCache()
		if msg.Err != nil {
			m.deps.Logger.ErrorContext(m.deps.Ctx, "historical data fetch failed",
				xslog.Source(msg.ErrSource),
				xslog.Error(msg.Err))
		} else {
			m.state.dashboard.Averages = dashboard.ComputeAverages(msg.Recoveries, msg.Cycles, msg.Sleeps)
			// store full recoveries for calendar coloring (50 days)
			m.state.dashboard.CalendarRecoveries = msg.Recoveries
			// store last 7 days of data for charts
			m.state.dashboard.HistoricalRecoveries = xslices.Truncate(msg.Recoveries, 7)
			m.state.dashboard.HistoricalCycles = xslices.Truncate(msg.Cycles, 7)
			m.state.dashboard.HistoricalSleeps = xslices.Truncate(msg.Sleeps, 7)
		}
		return m, nil

	case dashboard.CalendarRecoveriesMsg:
		m.state.dashboard.CalendarLoading = false
		if msg.Err == nil {
			// merge new recoveries with existing ones
			m.state.dashboard.CalendarRecoveries = cache.MergeRecoverySlices(m.state.dashboard.CalendarRecoveries, msg.Recoveries)
		}
		return m, nil

	case dashboard.CalendarSpinnerTickMsg:
		// only tick if still loading
		if m.state.dashboard.CalendarLoading && m.state.dashboard.CalendarMode {
			m.state.dashboard.CalendarSpinnerStep++
			return m, dashboard.CalendarSpinnerTickCmd()
		}
		return m, nil

	case NotificationMsg:
		if m.page == page.Dashboard {
			return m, tea.Batch(
				dashboard.FetchCycleCmd(m.deps.Ctx, m.deps.CacheService),
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
	// handle calendar mode separately
	if m.page == page.Dashboard && m.state.dashboard.CalendarMode {
		return m.handleCalendarKeyMsg(msg)
	}

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
			m.state.dashboard.ScrollOffset = 0 // reset scroll on tab change
			return m, nil
		}
	case "left", "h":
		if m.page == page.Dashboard {
			m.state.dashboard.ActiveTab = dashboard.PrevTab(m.state.dashboard.ActiveTab)
			m.state.dashboard.ScrollOffset = 0 // reset scroll on tab change
			return m, nil
		}
	case "[", "H":
		if m.page == page.Dashboard && m.state.dashboard.ActiveTab == dashboard.TabOverview {
			// Go back one day (no limit)
			current := m.state.dashboard.EffectiveDate()
			newDate := current.AddDate(0, 0, -1)
			return m, m.handleDateChange(&newDate)
		}
	case "]", "L":
		if m.page == page.Dashboard && m.state.dashboard.ActiveTab == dashboard.TabOverview {
			// Go forward one day only if not already at today
			current := m.state.dashboard.EffectiveDate()
			now := time.Now()

			if xtime.BeforeDay(current, now) {
				newDate := current.AddDate(0, 0, 1)
				if xtime.SameDay(newDate, now) {
					// reached today, set to nil (today mode)
					return m, m.handleDateChange(nil)
				}
				return m, m.handleDateChange(&newDate)
			}
			// Already at today, do nothing
			return m, nil
		}
	case "t":
		if m.page == page.Dashboard && m.state.dashboard.ActiveTab == dashboard.TabOverview {
			// Return to today
			if m.state.dashboard.SelectedDate != nil {
				return m, m.handleDateChange(nil)
			}
			return m, nil
		}
	case "c":
		if m.page == page.Dashboard && m.state.dashboard.ActiveTab == dashboard.TabOverview {
			// open calendar mode
			now := time.Now()
			effectiveDate := m.state.dashboard.EffectiveDate()
			m.state.dashboard.CalendarMode = true
			m.state.dashboard.CalendarCursor = effectiveDate
			m.state.dashboard.CalendarMonth = effectiveDate
			// ensure we don't have a future date as cursor
			if effectiveDate.After(now) {
				m.state.dashboard.CalendarCursor = now
				m.state.dashboard.CalendarMonth = now
			}
			return m, nil
		}
	case "j", "down":
		if m.page == page.Dashboard {
			if m.state.dashboard.ActiveTab == dashboard.TabOverview {
				m.state.dashboard.ActiveTab = dashboard.TabRecovery
			} else {
				// Scroll down in drill pages (clamping happens in View)
				m.state.dashboard.ScrollOffset++
			}
			return m, nil
		}
	case "k", "up":
		if m.page == page.Dashboard {
			if m.state.dashboard.ActiveTab == dashboard.TabOverview {
				m.state.dashboard.ActiveTab = dashboard.TabRecovery
			} else {
				// scroll up in drill pages
				if m.state.dashboard.ScrollOffset > 0 {
					m.state.dashboard.ScrollOffset--
				}
			}
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

func (m *Model) handleCalendarKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	now := time.Now()
	cal := calendar.New(m.state.dashboard.CalendarCursor, now)

	// update month view to match current state
	if m.state.dashboard.CalendarMonth.Month() != cal.Month().Month() ||
		m.state.dashboard.CalendarMonth.Year() != cal.Month().Year() {
		for cal.Month().Before(m.state.dashboard.CalendarMonth) {
			cal = cal.NextMonth()
		}
		for cal.Month().After(m.state.dashboard.CalendarMonth) {
			cal = cal.PrevMonth()
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		m.deps.Cancel()
		return m, tea.Quit
	case "esc":
		// close calendar without changing date
		m.state.dashboard.CalendarMode = false
		return m, nil
	case "enter":
		// select highlighted date and close calendar
		m.state.dashboard.CalendarMode = false
		selectedDate := cal.Cursor()
		if xtime.SameDay(selectedDate, now) {
			return m, m.handleDateChange(nil)
		}
		return m, m.handleDateChange(&selectedDate)
	case "t":
		// jump to today and select
		m.state.dashboard.CalendarMode = false
		return m, m.handleDateChange(nil)
	case "h", "left":
		// move cursor left (previous day)
		cal = cal.MoveCursor(-1)
		return m, m.updateCalendarPosition(cal)
	case "l", "right":
		// move cursor right (next day)
		cal = cal.MoveCursor(1)
		return m, m.updateCalendarPosition(cal)
	case "j", "down":
		// move cursor down (next week)
		cal = cal.MoveCursor(7)
		return m, m.updateCalendarPosition(cal)
	case "k", "up":
		// move cursor up (previous week)
		cal = cal.MoveCursor(-7)
		return m, m.updateCalendarPosition(cal)
	case "H", "[":
		// previous month
		cal = cal.PrevMonth()
		return m, m.updateCalendarPosition(cal)
	case "L", "]":
		// next month (clamped to current month)
		cal = cal.NextMonth()
		return m, m.updateCalendarPosition(cal)
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
			dashboard.FetchCycleCmd(m.deps.Ctx, m.deps.CacheService),
			dashboard.FetchHistoricalDataCmd(m.deps.Ctx, m.deps.CacheService),
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
		m.deps.CacheService.SetAPIKey(msg.APIKey)
		m.deps.SSEClient.SetAPIKey(msg.APIKey)
	}

	m.state.dashboard.AuthIndicator.Authenticated = true
	m.page = page.Dashboard
	return m, tea.Batch(
		dashboard.FetchCycleCmd(m.deps.Ctx, m.deps.CacheService),
		dashboard.FetchHistoricalDataCmd(m.deps.Ctx, m.deps.CacheService),
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

func (m *Model) handleDateChange(date *time.Time) tea.Cmd {
	// clear chart cache when navigating dates
	chart.ClearCache()

	m.state.dashboard.SelectedDate = date
	m.state.dashboard.SetPending()

	m.state.dashboard.CurrentCycle = nil
	m.state.dashboard.CurrentSleep = nil
	m.state.dashboard.CurrentRecovery = nil
	m.state.dashboard.SleepScore = nil
	m.state.dashboard.RecoveryScore = nil
	m.state.dashboard.StrainScore = nil
	m.state.dashboard.TodaysWorkouts = nil
	m.state.dashboard.Averages = nil
	m.state.dashboard.CalendarRecoveries = nil
	m.state.dashboard.HistoricalRecoveries = nil
	m.state.dashboard.HistoricalCycles = nil
	m.state.dashboard.HistoricalSleeps = nil

	refDate := m.state.dashboard.EffectiveDate()
	return tea.Batch(
		dashboard.FetchCycleForDateCmd(m.deps.Ctx, m.deps.CacheService, refDate),
		dashboard.FetchHistoricalDataForDateCmd(m.deps.Ctx, m.deps.CacheService, refDate),
	)
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
		gauges := dashboard.View(&m.state.dashboard, m.viewportWidth, m.viewportHeight)

		f := footer.New(dashboard.AuthIndicatorView(m.state.dashboard), m.viewportWidth)

		switch m.state.dashboard.ActiveTab {
		case dashboard.TabOverview:
			if m.state.dashboard.CalendarMode {
				f = f.WithNavHints("←↑↓→ navigate    enter select    esc cancel    t today")
			} else {
				f = f.WithNavHints("[/] date    c calendar    ← sleep    ↑↓ recovery    → strain")
			}
		default:
			f = f.WithNavHints("esc back    ←/→ navigate    ↑/↓ scroll")
		}

		footerOverlay := lipgloss.Place(
			m.viewportWidth,
			m.viewportHeight,
			lipgloss.Left,
			lipgloss.Bottom,
			f.Render(),
		)

		// Render calendar overlay when in calendar mode (replaces gauges entirely)
		if m.state.dashboard.CalendarMode {
			cal := calendar.New(m.state.dashboard.CalendarCursor, time.Now())
			// Sync month view
			for cal.Month().Month() != m.state.dashboard.CalendarMonth.Month() ||
				cal.Month().Year() != m.state.dashboard.CalendarMonth.Year() {
				if cal.Month().Before(m.state.dashboard.CalendarMonth) {
					cal = cal.NextMonth()
				} else {
					cal = cal.PrevMonth()
				}
			}
			recoveryData := calendar.BuildRecoveryData(m.state.dashboard.CalendarRecoveries)
			cal = cal.WithRecoveryData(recoveryData).
				WithLoading(m.state.dashboard.CalendarLoading).
				WithSpinnerStep(m.state.dashboard.CalendarSpinnerStep)
			calendarContent := cal.Render()
			calendarView := lipgloss.Place(
				m.viewportWidth,
				m.viewportHeight,
				lipgloss.Center,
				lipgloss.Center,
				calendarContent,
			)
			content = m.overlayStrings(calendarView, footerOverlay)
		} else {
			content = m.overlayStrings(gauges, footerOverlay)
		}
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

// updateCalendarPosition updates the calendar position, fetching data if month changed.
// CacheService handles staleness checking - we always call fetch and let it decide
// whether to return cached data or fetch from API.
func (m *Model) updateCalendarPosition(cal calendar.Calendar) tea.Cmd {
	newCursor := cal.Cursor()
	newMonth := cal.Month()
	oldMonth := m.state.dashboard.CalendarMonth

	// always update position immediately
	m.state.dashboard.CalendarCursor = newCursor
	m.state.dashboard.CalendarMonth = newMonth

	monthChanged := newMonth.Month() != oldMonth.Month() || newMonth.Year() != oldMonth.Year()

	if monthChanged {
		m.state.dashboard.CalendarLoading = true
		m.state.dashboard.CalendarSpinnerStep = 0
		return tea.Batch(
			dashboard.FetchCalendarRecoveriesCmd(m.deps.Ctx, m.deps.CacheService, newMonth),
			dashboard.CalendarSpinnerTickCmd(),
		)
	}

	return nil
}
