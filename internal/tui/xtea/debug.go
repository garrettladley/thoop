package xtea

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// DebugBorder wraps content with a colored border for debugging layout issues.
func DebugBorder(content string, borderColor color.Color) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor)
	return style.Render(content)
}
