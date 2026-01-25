package chart

import (
	"image/color"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// ColorFunc returns a color based on a value.
type ColorFunc func(value float64) color.Color

// RecoveryColor returns green/yellow/red based on recovery percentage.
func RecoveryColor(value float64) color.Color {
	switch {
	case value >= 67:
		return theme.ColorHighRecovery
	case value >= 34:
		return theme.ColorMediumRecovery
	default:
		return theme.ColorLowRecovery
	}
}

// StrainColor returns the strain color (blue).
func StrainColor(_ float64) color.Color {
	return theme.ColorStrain
}

// SleepColor returns the sleep color.
func SleepColor(_ float64) color.Color {
	return theme.ColorSleep
}

// SleepPerformanceColor returns colors based on sleep performance percentage.
func SleepPerformanceColor(value float64) color.Color {
	switch {
	case value >= 85:
		return theme.ColorTeal // optimal
	case value >= 70:
		return theme.ColorNeutral // sufficient
	default:
		return theme.ColorOrange // poor
	}
}

// StaticColor returns a ColorFunc that always returns the given color.
func StaticColor(c color.Color) ColorFunc {
	return func(_ float64) color.Color {
		return c
	}
}
