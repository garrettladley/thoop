//go:build !release

package footer

import (
	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
	"github.com/garrettladley/thoop/internal/version"
)

var devVersionStyle = lipgloss.NewStyle().Foreground(theme.ColorDim)

// leftContent returns the version string for dev builds.
func (f Footer) leftContent() string {
	return devVersionStyle.Render(version.Get())
}
