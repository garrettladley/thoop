package footer

import "charm.land/lipgloss/v2"

type Footer struct {
	rightContent string
	width        int
	padding      int
}

func New(rightContent string, width int) Footer {
	return Footer{
		rightContent: rightContent,
		width:        width,
		padding:      2,
	}
}

func (f Footer) Render() string {
	leftContent := f.leftContent()

	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(f.rightContent)
	spacerWidth := max(f.width-leftWidth-rightWidth-(f.padding*2), 0)

	spacer := make([]byte, spacerWidth)
	for i := range spacer {
		spacer[i] = ' '
	}

	return lipgloss.NewStyle().
		PaddingLeft(f.padding).
		PaddingRight(f.padding).
		PaddingBottom(1).
		Render(leftContent + string(spacer) + f.rightContent)
}
