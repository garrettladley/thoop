package chart

import (
	"fmt"
	"image/color"
)

// DataPoint represents a single data point with a label and value.
type DataPoint struct {
	Label string
	Value float64
}

// Series represents a named series of data points with a color.
type Series struct {
	Name   string
	Points []DataPoint
	Color  color.Color
}

// StackedDataPoint represents a data point with multiple stacked values.
type StackedDataPoint struct {
	Label  string
	Values []float64 // values for each segment
}

// ValueFormatter formats a float64 value for display.
type ValueFormatter func(value float64) string

// FormatPercentage formats a value as a percentage.
func FormatPercentage(v float64) string {
	return fmt.Sprintf("%.0f%%", v)
}

// FormatInt formats a value as an integer.
func FormatInt(v float64) string {
	return fmt.Sprintf("%.0f", v)
}

// FormatIntWithCommas formats a value as an integer with comma separators.
func FormatIntWithCommas(v float64) string {
	n := int64(v)
	if n < 0 {
		return "-" + formatWithCommas(-n)
	}
	return formatWithCommas(n)
}

func formatWithCommas(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return formatWithCommas(n/1000) + fmt.Sprintf(",%03d", n%1000)
}

// FormatDuration formats milliseconds as "H:MM".
func FormatDuration(ms float64) string {
	totalMinutes := int(ms / (1000 * 60))
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	return fmt.Sprintf("%d:%02d", hours, minutes)
}

// FormatDurationFromHours formats hours as "H:MM".
func FormatDurationFromHours(hours float64) string {
	h := int(hours)
	m := int((hours - float64(h)) * 60)
	return fmt.Sprintf("%d:%02d", h, m)
}

// FormatFloat1 formats a value with one decimal place.
func FormatFloat1(v float64) string {
	return fmt.Sprintf("%.1f", v)
}
