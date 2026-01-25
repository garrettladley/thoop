package date_header

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/xtime"
)

// Render renders the date header centered in the given width.
// If selectedDate is nil or is today, displays "TODAY".
// Otherwise displays the date as "Mon, Jan 2".
func Render(selectedDate *time.Time, width int) string {
	var label string
	if selectedDate == nil || xtime.IsToday(*selectedDate) {
		label = "TODAY"
	} else {
		label = selectedDate.Format("Mon, Jan 2")
	}

	style := lipgloss.NewStyle().
		Foreground(theme.ColorDim)

	styledLabel := style.Render(label)
	labelWidth := lipgloss.Width(styledLabel)

	leftPadding := (width - labelWidth) / 2
	leftPadding = max(leftPadding, 0)

	return "\n" + strings.Repeat(" ", leftPadding) + styledLabel
}
