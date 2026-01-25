package dateheader

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/xtime"
)

// Render renders the date header centered-left in the given width.
// If selectedDate is nil or is today, displays "TODAY".
// Otherwise displays the date as "Mon Jan 2".
func Render(selectedDate *time.Time, width int) string {
	var label string
	if selectedDate == nil || xtime.IsToday(*selectedDate) {
		label = "TODAY"
	} else {
		label = selectedDate.Format("Mon Jan 2")
	}

	style := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Bold(true)

	styledLabel := style.Render(label)
	labelWidth := lipgloss.Width(styledLabel)

	// Position at 1/3 of width (centered-left)
	leftPadding := width/3 - labelWidth/2
	if leftPadding < 0 {
		leftPadding = 0
	}

	return strings.Repeat(" ", leftPadding) + styledLabel
}
