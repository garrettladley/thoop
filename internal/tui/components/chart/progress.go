package chart

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// HorizontalProgress renders a horizontal progress bar with labels.
type HorizontalProgress struct {
	actual      float64
	target      float64
	maxValue    float64
	actualLabel string
	targetLabel string
	actualColor color.Color
	targetColor color.Color
	bgColor     color.Color
	formatter   ValueFormatter
	showLabels  bool
}

// HorizontalProgressOption configures a HorizontalProgress.
type HorizontalProgressOption func(*HorizontalProgress)

// WithProgressLabels sets the labels for actual and target.
func WithProgressLabels(actual, target string) HorizontalProgressOption {
	return func(hp *HorizontalProgress) {
		hp.actualLabel = actual
		hp.targetLabel = target
	}
}

// WithProgressColors sets the colors for actual and target bars.
func WithProgressColors(actual, target color.Color) HorizontalProgressOption {
	return func(hp *HorizontalProgress) {
		hp.actualColor = actual
		hp.targetColor = target
	}
}

// WithProgressBgColor sets the background color.
func WithProgressBgColor(c color.Color) HorizontalProgressOption {
	return func(hp *HorizontalProgress) {
		hp.bgColor = c
	}
}

// WithProgressFormatter sets the value formatter.
func WithProgressFormatter(f ValueFormatter) HorizontalProgressOption {
	return func(hp *HorizontalProgress) {
		hp.formatter = f
	}
}

// WithProgressShowLabels shows or hides labels.
func WithProgressShowLabels(show bool) HorizontalProgressOption {
	return func(hp *HorizontalProgress) {
		hp.showLabels = show
	}
}

// NewHorizontalProgress creates a new horizontal progress component.
// actual and target are the values to display, maxValue is the scale maximum.
func NewHorizontalProgress(actual, target, maxValue float64, opts ...HorizontalProgressOption) *HorizontalProgress {
	hp := &HorizontalProgress{
		actual:      actual,
		target:      target,
		maxValue:    maxValue,
		actualLabel: "ACTUAL",
		targetLabel: "TARGET",
		actualColor: theme.ColorSleep,
		targetColor: theme.ColorDim,
		bgColor:     theme.ColorBgLight,
		formatter:   FormatDurationFromHours,
		showLabels:  true,
	}
	for _, opt := range opts {
		opt(hp)
	}
	return hp
}

// Render renders the horizontal progress.
func (hp *HorizontalProgress) Render(width int) string {
	if width <= 0 {
		return ""
	}

	var sections []string

	labelWidth := 0
	if hp.showLabels {
		labelWidth = max(len(hp.actualLabel), len(hp.targetLabel)) + 2
	}

	barWidth := width - labelWidth - 10 // 10 for value display
	if barWidth < 10 {
		barWidth = 10
	}

	// render actual bar
	actualRow := hp.renderProgressRow(hp.actualLabel, hp.actual, hp.actualColor, labelWidth, barWidth)
	sections = append(sections, actualRow)

	// render target bar
	targetRow := hp.renderProgressRow(hp.targetLabel, hp.target, hp.targetColor, labelWidth, barWidth)
	sections = append(sections, targetRow)

	return strings.Join(sections, "\n")
}

func (hp *HorizontalProgress) renderProgressRow(label string, value float64, c color.Color, labelWidth, barWidth int) string {
	var result strings.Builder

	if hp.showLabels {
		labelStyle := lipgloss.NewStyle().Foreground(theme.ColorDim).Width(labelWidth)
		result.WriteString(labelStyle.Render(label))
	}

	fillPct := value / hp.maxValue
	if fillPct > 1 {
		fillPct = 1
	}
	if fillPct < 0 {
		fillPct = 0
	}

	bar := NewHorizontalBar(barWidth, []Segment{{Value: fillPct, Color: c}},
		WithHorizontalBarBgColor(hp.bgColor))
	result.WriteString(bar.Render())

	result.WriteString(" ")
	valueStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)
	result.WriteString(valueStyle.Render(hp.formatter(value)))

	return result.String()
}

// ComparisonProgress renders two horizontal bars for comparison.
type ComparisonProgress struct {
	values    []ProgressValue
	maxValue  float64
	bgColor   color.Color
	formatter ValueFormatter
}

// ProgressValue represents a single progress bar with label.
type ProgressValue struct {
	Label string
	Value float64
	Color color.Color
}

// ComparisonProgressOption configures a ComparisonProgress.
type ComparisonProgressOption func(*ComparisonProgress)

// WithComparisonBgColor sets the background color.
func WithComparisonBgColor(c color.Color) ComparisonProgressOption {
	return func(cp *ComparisonProgress) {
		cp.bgColor = c
	}
}

// WithComparisonFormatter sets the value formatter.
func WithComparisonFormatter(f ValueFormatter) ComparisonProgressOption {
	return func(cp *ComparisonProgress) {
		cp.formatter = f
	}
}

// NewComparisonProgress creates a new comparison progress component.
func NewComparisonProgress(values []ProgressValue, maxValue float64, opts ...ComparisonProgressOption) *ComparisonProgress {
	cp := &ComparisonProgress{
		values:    values,
		maxValue:  maxValue,
		bgColor:   theme.ColorBgLight,
		formatter: FormatInt,
	}
	for _, opt := range opts {
		opt(cp)
	}
	return cp
}

// Render renders the comparison progress bars.
func (cp *ComparisonProgress) Render(width int) string {
	if len(cp.values) == 0 || width <= 0 {
		return ""
	}

	// find max label width
	labelWidth := 0
	for _, v := range cp.values {
		if len(v.Label) > labelWidth {
			labelWidth = len(v.Label)
		}
	}
	labelWidth += 2 // padding

	barWidth := width - labelWidth - 10
	if barWidth < 10 {
		barWidth = 10
	}

	lines := make([]string, 0, len(cp.values))
	for _, v := range cp.values {
		var row strings.Builder

		labelStyle := lipgloss.NewStyle().Foreground(theme.ColorDim).Width(labelWidth)
		row.WriteString(labelStyle.Render(v.Label))

		fillPct := v.Value / cp.maxValue
		if fillPct > 1 {
			fillPct = 1
		}
		if fillPct < 0 {
			fillPct = 0
		}

		bar := NewHorizontalBar(barWidth, []Segment{{Value: fillPct, Color: v.Color}},
			WithHorizontalBarBgColor(cp.bgColor))
		row.WriteString(bar.Render())

		row.WriteString(" ")
		valueStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)
		row.WriteString(valueStyle.Render(cp.formatter(v.Value)))

		lines = append(lines, row.String())
	}

	return strings.Join(lines, "\n")
}
