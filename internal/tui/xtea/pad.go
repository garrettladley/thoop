package xtea

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// PadLinesToWidth pads each line in sections to fill the given width.
func PadLinesToWidth(sections []string, width int) {
	for i, line := range sections {
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			sections[i] = line + strings.Repeat(" ", width-lineWidth)
		}
	}
}
