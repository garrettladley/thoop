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

// SleepPerformanceColor returns the sleep performance color.
func SleepPerformanceColor(_ float64) color.Color {
	return color.RGBA{R: 131, G: 160, B: 184, A: 255}
}

// StaticColor returns a ColorFunc that always returns the given color.
func StaticColor(c color.Color) ColorFunc {
	return func(_ float64) color.Color {
		return c
	}
}

// SleepConsistencyColor returns color based on sleep consistency percentage.
// Optimal: 80+, Sufficient: 70-79, Poor: <70
func SleepConsistencyColor(value float64) color.Color {
	switch {
	case value >= 80:
		return theme.ColorTeal
	case value >= 70:
		return theme.ColorNeutral
	default:
		return theme.ColorOrange
	}
}

// HoursVsNeededColor returns color based on hours vs needed percentage.
// Optimal: 85+, Sufficient: 70-85, Poor: <70
func HoursVsNeededColor(value float64) color.Color {
	switch {
	case value >= 85:
		return theme.ColorTeal
	case value >= 70:
		return theme.ColorNeutral
	default:
		return theme.ColorOrange
	}
}

// SleepEfficiencyColor returns color based on sleep efficiency percentage.
// Optimal: 90+, Sufficient: 80-89, Poor: <80
func SleepEfficiencyColor(value float64) color.Color {
	switch {
	case value >= 90:
		return theme.ColorTeal
	case value >= 80:
		return theme.ColorNeutral
	default:
		return theme.ColorOrange
	}
}
