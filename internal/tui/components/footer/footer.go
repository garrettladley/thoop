package footer

import (
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

type Footer struct {
	rightContent  string
	centerContent string
	width         int
	padding       int
}

func New(rightContent string, width int) Footer {
	return Footer{
		rightContent: rightContent,
		width:        width,
		padding:      2,
	}
}

func (f Footer) WithNavHints(hint string) Footer {
	hintStyle := lipgloss.NewStyle().Foreground(theme.ColorNavHint)
	f.centerContent = hintStyle.Render(hint)
	return f
}

func (f Footer) Render() string {
	leftContent := f.leftContent()

	// true center of screen
	centerWidth := lipgloss.Width(f.centerContent)
	centerStart := (f.width - centerWidth) / 2

	leftWidth := lipgloss.Width(leftContent) + f.padding
	rightWidth := lipgloss.Width(f.rightContent) + f.padding

	// gap b/w left content and center content
	leftToCenterGap := max(centerStart-leftWidth, 1)

	// gap b/w center content and right content
	centerEnd := centerStart + centerWidth
	centerToRightGap := max(f.width-centerEnd-rightWidth, 1)

	leftSpacer := make([]byte, leftToCenterGap)
	for i := range leftSpacer {
		leftSpacer[i] = ' '
	}

	rightSpacer := make([]byte, centerToRightGap)
	for i := range rightSpacer {
		rightSpacer[i] = ' '
	}

	row := leftContent + string(leftSpacer) + f.centerContent + string(rightSpacer) + f.rightContent

	return lipgloss.NewStyle().
		PaddingLeft(f.padding).
		PaddingRight(f.padding).
		PaddingBottom(1).
		Render(row)
}
