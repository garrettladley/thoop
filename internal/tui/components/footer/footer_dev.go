//go:build dev

package footer

import (
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop"
	"github.com/garrettladley/thoop/internal/tui/theme"
)

var devVersionStyle = lipgloss.NewStyle().Foreground(theme.ColorDim)

func (f Footer) leftContent() string {
	return devVersionStyle.Render(thoop.Version)
}
