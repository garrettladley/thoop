package calendar

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/xtime"
)

type RecoveryData map[string]float64

func BuildRecoveryData(recoveries []whoop.Recovery, cycles []whoop.Cycle) RecoveryData {
	// build cycleID -> start date map
	cycleStarts := make(map[int64]time.Time, len(cycles))
	for _, c := range cycles {
		cycleStarts[c.ID] = c.Start
	}

	data := make(RecoveryData, len(recoveries))
	for _, r := range recoveries {
		if r.Score != nil {
			// use cycle start date if available, fall back to CreatedAt
			dateKey := r.CreatedAt.Format("2006-01-02")
			if cycleStart, ok := cycleStarts[r.CycleID]; ok {
				dateKey = cycleStart.Format("2006-01-02")
			}
			data[dateKey] = r.Score.RecoveryScore
		}
	}
	return data
}

const (
	colWidth  = 6 // width per day column
	numCols   = 7 // days per week
	gridWidth = colWidth*numCols + 2
	minWidth  = 50
	minHeight = 15
)

var dayHeaders = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// calendar renders a calendar modal for date selection.
type Calendar struct {
	month        time.Time    // month being viewed (any day in that month)
	cursor       time.Time    // currently highlighted date
	today        time.Time    // current date (for future date restrictions)
	recoveryData RecoveryData // optional recovery scores for coloring dates
	loading      bool         // true when fetching recovery data
	spinnerStep  int          // current spinner animation frame
}

// spinner frames for loading animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// New creates a calendar starting on the given date.
func New(cursor time.Time, today time.Time) Calendar {
	return Calendar{
		month:  cursor,
		cursor: cursor,
		today:  today,
	}
}

// Cursor returns the currently highlighted date.
func (c Calendar) Cursor() time.Time {
	return c.cursor
}

// Month returns the currently viewed month.
func (c Calendar) Month() time.Time {
	return c.month
}

// MoveCursor moves the cursor by the given number of days.
// Returns updated calendar. Does not move past today.
func (c Calendar) MoveCursor(days int) Calendar {
	newCursor := c.cursor.AddDate(0, 0, days)

	// don't allow moving to future dates
	if newCursor.After(c.today) {
		return c
	}

	c.cursor = newCursor

	// update month view if cursor moves to different month
	if newCursor.Month() != c.month.Month() || newCursor.Year() != c.month.Year() {
		c.month = newCursor
	}

	return c
}

// PrevMonth moves to the previous month, keeping cursor on same day if possible.
func (c Calendar) PrevMonth() Calendar {
	c.month = c.month.AddDate(0, -1, 0)

	// move cursor to same day in new month, or last day of month
	newCursor := time.Date(c.month.Year(), c.month.Month(), c.cursor.Day(), 0, 0, 0, 0, c.cursor.Location())

	// clamp to last day of month if day doesn't exist
	lastDay := xtime.LastDayOfMonth(c.month)
	if newCursor.Day() > lastDay.Day() {
		newCursor = lastDay
	}

	c.cursor = newCursor
	return c
}

// NextMonth moves to the next month if it wouldn't go past current month.
func (c Calendar) NextMonth() Calendar {
	nextMonth := c.month.AddDate(0, 1, 0)

	// don't allow navigating past current month
	if nextMonth.Year() > c.today.Year() ||
		(nextMonth.Year() == c.today.Year() && nextMonth.Month() > c.today.Month()) {
		return c
	}

	c.month = nextMonth

	// move cursor to same day in new month, or last day of month
	newCursor := time.Date(c.month.Year(), c.month.Month(), c.cursor.Day(), 0, 0, 0, 0, c.cursor.Location())

	// clamp to last day of month if day doesn't exist
	lastDay := xtime.LastDayOfMonth(c.month)
	if newCursor.Day() > lastDay.Day() {
		newCursor = lastDay
	}

	// don't allow cursor past today
	if newCursor.After(c.today) {
		newCursor = c.today
	}

	c.cursor = newCursor
	return c
}

// JumpToToday moves cursor and month view to today.
func (c Calendar) JumpToToday() Calendar {
	c.cursor = c.today
	c.month = c.today
	return c
}

// WithRecoveryData returns a calendar with recovery data for coloring dates.
func (c Calendar) WithRecoveryData(data RecoveryData) Calendar {
	c.recoveryData = data
	return c
}

// WithLoading returns a calendar with the loading state set.
func (c Calendar) WithLoading(loading bool) Calendar {
	c.loading = loading
	return c
}

// WithSpinnerStep returns a calendar with the spinner step set.
func (c Calendar) WithSpinnerStep(step int) Calendar {
	c.spinnerStep = step
	return c
}

// Render renders the calendar as a string.
func (c Calendar) Render() string {
	var b strings.Builder

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorDim).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	dayStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Width(colWidth).
		Align(lipgloss.Center)

	futureDayStyle := dayStyle.Foreground(theme.ColorDim)

	paddingDayStyle := dayStyle.Foreground(lipgloss.Color("#333333"))

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.ColorNavHint)

	title := c.month.Format("January")
	if c.month.Year() != c.today.Year() {
		title = c.month.Format("January 2006")
	}

	titleWidth := gridWidth
	titlePadding := max((titleWidth-len(title))/2, 0)
	b.WriteString(strings.Repeat(" ", titlePadding))
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	for i, day := range dayHeaders {
		b.WriteString(headerStyle.Width(colWidth).Align(lipgloss.Center).Render(day))
		if i < len(dayHeaders)-1 {
			b.WriteString("")
		}
	}
	b.WriteString("\n\n")

	// get first day of month and what weekday it starts on
	firstOfMonth := time.Date(c.month.Year(), c.month.Month(), 1, 0, 0, 0, 0, c.month.Location())
	startWeekday := int(firstOfMonth.Weekday()) // 0 = Sunday

	// get days in current month
	lastOfMonth := xtime.LastDayOfMonth(c.month)
	daysInMonth := lastOfMonth.Day()

	// get days from previous month for padding
	prevMonth := c.month.AddDate(0, -1, 0)
	lastOfPrevMonth := xtime.LastDayOfMonth(prevMonth)

	dayNum := 1
	nextMonthDay := 1

	// actual grid width is 7 columns * colWidth each
	actualGridWidth := numCols * colWidth

	// always render 6 weeks to prevent layout shift between months
	if c.loading {
		// render loading spinner centered in grid
		spinnerStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)
		frame := spinnerFrames[c.spinnerStep%len(spinnerFrames)]
		spinner := spinnerStyle.Render(frame)
		spinnerWidth := lipgloss.Width(spinner)

		// place spinner in middle row (row 3)
		for week := range 6 {
			if week == 2 {
				// center spinner in this row
				leftPad := (actualGridWidth - spinnerWidth) / 2
				rightPad := actualGridWidth - leftPad - spinnerWidth
				b.WriteString(strings.Repeat(" ", leftPad))
				b.WriteString(spinner)
				b.WriteString(strings.Repeat(" ", rightPad))
			} else {
				// empty row - same width as day grid
				b.WriteString(strings.Repeat(" ", actualGridWidth))
			}
			b.WriteString("\n\n")
		}
	} else {
		for week := range 6 {
			for weekday := range 7 {
				cellIndex := week*7 + weekday

				rendered, newDayNum, newNextMonthDay := c.renderCell(
					cellIndex, startWeekday, dayNum, daysInMonth, nextMonthDay,
					lastOfPrevMonth, paddingDayStyle, dayStyle, futureDayStyle,
				)
				dayNum = newDayNum
				nextMonthDay = newNextMonthDay

				b.WriteString(rendered)
			}
			b.WriteString("\n\n")
		}
	}

	// render legend
	legend := renderLegend()
	legendPadding := max((gridWidth-lipgloss.Width(legend))/2, 0)
	b.WriteString(strings.Repeat(" ", legendPadding))
	b.WriteString(legend)
	b.WriteString("\n\n")

	hints := "[/H prev    L/] next"
	hintPadding := max((gridWidth-len(hints))/2, 0)
	b.WriteString(strings.Repeat(" ", hintPadding))
	b.WriteString(hintStyle.Render(hints))

	return borderStyle.Render(b.String())
}

// renderCell returns the rendered cell string and updated counters for a calendar cell.
func (c Calendar) renderCell(
	cellIndex, startWeekday, dayNum, daysInMonth, nextMonthDay int,
	lastOfPrevMonth time.Time,
	paddingDayStyle, dayStyle, futureDayStyle lipgloss.Style,
) (string, int, int) {
	// previous month padding
	if cellIndex < startWeekday {
		prevDay := lastOfPrevMonth.Day() - (startWeekday - cellIndex - 1)
		return paddingDayStyle.Render(fmt.Sprintf("%d", prevDay)), dayNum, nextMonthDay
	}

	// next month padding
	if dayNum > daysInMonth {
		return paddingDayStyle.Render(fmt.Sprintf("%d", nextMonthDay)), dayNum, nextMonthDay + 1
	}

	// current month day
	currentDate := time.Date(c.month.Year(), c.month.Month(), dayNum, 0, 0, 0, 0, c.month.Location())
	isCursor := xtime.SameDay(currentDate, c.cursor)

	// determine style for the day number
	numStyle := futureDayStyle // default to dimmed
	if currentDate.After(c.today) {
		// future dates stay dimmed
		numStyle = futureDayStyle
	} else if c.recoveryData != nil {
		// past dates: color by recovery score if available, otherwise dimmed
		dateKey := currentDate.Format("2006-01-02")
		if score, ok := c.recoveryData[dateKey]; ok {
			numStyle = c.recoveryStyle(score, dayStyle)
		}
		// no data = stays dimmed (futureDayStyle)
	}
	// if no recovery data at all, stays dimmed

	// render cell with optional cursor brackets
	dayStr := fmt.Sprintf("%d", dayNum)
	if isCursor {
		// brackets in white, number in recovery color (without width constraint)
		bracketStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)
		numOnlyStyle := numStyle.Width(0) // remove width to avoid extra padding
		rendered := bracketStyle.Render("[") + numOnlyStyle.Render(dayStr) + bracketStyle.Render("]")
		// center within column width
		width := lipgloss.Width(rendered)
		if width < colWidth {
			pad := (colWidth - width) / 2
			rendered = strings.Repeat(" ", pad) + rendered + strings.Repeat(" ", colWidth-width-pad)
		}
		return rendered, dayNum + 1, nextMonthDay
	}

	return numStyle.Render(dayStr), dayNum + 1, nextMonthDay
}

// recoveryStyle returns a style colored by recovery score.
func (c Calendar) recoveryStyle(score float64, baseStyle lipgloss.Style) lipgloss.Style {
	switch {
	case score >= 67:
		return baseStyle.Foreground(theme.ColorHighRecovery)
	case score >= 34:
		return baseStyle.Foreground(theme.ColorMediumRecovery)
	default:
		return baseStyle.Foreground(theme.ColorLowRecovery)
	}
}

// renderLegend renders the recovery color legend.
func renderLegend() string {
	lowStyle := lipgloss.NewStyle().Foreground(theme.ColorLowRecovery)
	medStyle := lipgloss.NewStyle().Foreground(theme.ColorMediumRecovery)
	highStyle := lipgloss.NewStyle().Foreground(theme.ColorHighRecovery)
	dimStyle := lipgloss.NewStyle().Foreground(theme.ColorDim)

	return fmt.Sprintf("%s %s  %s %s  %s %s",
		lowStyle.Render(theme.SymbolCircleFilled),
		dimStyle.Render("<34%"),
		medStyle.Render(theme.SymbolCircleFilled),
		dimStyle.Render("34-66%"),
		highStyle.Render(theme.SymbolCircleFilled),
		dimStyle.Render(">66%"),
	)
}
