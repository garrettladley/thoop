package calendar

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/xtime"
)

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
	month  time.Time // month being viewed (any day in that month)
	cursor time.Time // currently highlighted date
	today  time.Time // current date (for future date restrictions)
}

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

	cursorStyle := lipgloss.NewStyle().
		Foreground(theme.ColorBlack).
		Background(theme.ColorWhite).
		Width(colWidth).
		Align(lipgloss.Center)

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

	// always render 6 weeks to prevent layout shift between months
	for week := range 6 {
		for weekday := range 7 {
			cellIndex := week*7 + weekday

			cellStr, style, newDayNum, newNextMonthDay := c.renderCell(
				cellIndex, startWeekday, dayNum, daysInMonth, nextMonthDay,
				lastOfPrevMonth, paddingDayStyle, dayStyle, futureDayStyle, cursorStyle,
			)
			dayNum = newDayNum
			nextMonthDay = newNextMonthDay

			b.WriteString(style.Render(cellStr))
		}
		b.WriteString("\n\n")
	}

	hints := "[/H prev    L/] next"
	hintPadding := max((gridWidth-len(hints))/2, 0)
	b.WriteString(strings.Repeat(" ", hintPadding))
	b.WriteString(hintStyle.Render(hints))

	return borderStyle.Render(b.String())
}

// renderCell returns the display string, style, and updated counters for a calendar cell.
func (c Calendar) renderCell(
	cellIndex, startWeekday, dayNum, daysInMonth, nextMonthDay int,
	lastOfPrevMonth time.Time,
	paddingDayStyle, dayStyle, futureDayStyle, cursorStyle lipgloss.Style,
) (string, lipgloss.Style, int, int) {
	// previous month padding
	if cellIndex < startWeekday {
		prevDay := lastOfPrevMonth.Day() - (startWeekday - cellIndex - 1)
		return fmt.Sprintf("%d", prevDay), paddingDayStyle, dayNum, nextMonthDay
	}

	// next month padding
	if dayNum > daysInMonth {
		cellStr := fmt.Sprintf("%d", nextMonthDay)
		return cellStr, paddingDayStyle, dayNum, nextMonthDay + 1
	}

	// current month day
	cellStr := fmt.Sprintf("%d", dayNum)
	currentDate := time.Date(c.month.Year(), c.month.Month(), dayNum, 0, 0, 0, 0, c.month.Location())

	style := dayStyle
	if xtime.SameDay(currentDate, c.cursor) {
		style = cursorStyle
	} else if currentDate.After(c.today) {
		style = futureDayStyle
	}

	return cellStr, style, dayNum + 1, nextMonthDay
}
