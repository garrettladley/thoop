package chart

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// SleepStage represents a single sleep stage data.
type SleepStage struct {
	Name       string
	DurationMs int     // actual duration in milliseconds
	Percentage float64 // percentage of total sleep
	Color      color.Color
}

// SleepStagesChart renders a sleep stages breakdown chart.
type SleepStagesChart struct {
	id               string // unique identifier for caching
	stages           []SleepStage
	totalDuration    int // total duration in milliseconds
	showHeader       bool
	baselineDuration int // baseline duration in milliseconds for trend comparison
}

// SleepStagesChartOption configures a SleepStagesChart.
type SleepStagesChartOption func(*SleepStagesChart)

// WithSleepStagesBaseline sets the baseline duration for trend comparison.
func WithSleepStagesBaseline(baselineMs int) SleepStagesChartOption {
	return func(ssc *SleepStagesChart) {
		ssc.baselineDuration = baselineMs
	}
}

// WithSleepStagesID sets the chart ID for caching.
func WithSleepStagesID(id string) SleepStagesChartOption {
	return func(ssc *SleepStagesChart) {
		ssc.id = id
	}
}

// NewSleepStagesChart creates a new sleep stages chart.
func NewSleepStagesChart(stages []SleepStage, totalDuration int, opts ...SleepStagesChartOption) *SleepStagesChart {
	ssc := &SleepStagesChart{
		stages:        stages,
		totalDuration: totalDuration,
		showHeader:    true,
	}
	for _, opt := range opts {
		opt(ssc)
	}
	return ssc
}

// Render renders the sleep stages chart.
func (ssc *SleepStagesChart) Render(width int) string {
	if len(ssc.stages) == 0 || width <= 0 {
		return ""
	}

	if ssc.id != "" {
		dataHash := HashSleepStages(ssc.stages, ssc.totalDuration, ssc.baselineDuration)
		if cached, ok := GetCached(ssc.id, width, dataHash); ok {
			return cached
		}
		rendered := ssc.renderInternal(width)
		SetCached(ssc.id, width, dataHash, rendered)
		return rendered
	}

	return ssc.renderInternal(width)
}

// renderInternal performs the actual rendering without caching.
func (ssc *SleepStagesChart) renderInternal(width int) string {
	// pre-allocate: header (2 lines) + stages (2 lines each)
	sections := make([]string, 0, 2+len(ssc.stages)*2)

	if ssc.showHeader {
		sections = ssc.renderHeader(sections, width)
	}

	for _, stage := range ssc.stages {
		stageRow := ssc.renderStageRow(stage, width)
		sections = append(sections, stageRow)
	}

	return strings.Join(sections, "\n")
}

// renderHeader renders the header section with title and trend indicator.
func (ssc *SleepStagesChart) renderHeader(sections []string, width int) []string {
	labelStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite).Bold(true)
	durationStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite).Bold(true)

	leftLabel := labelStyle.Render("HOURS OF SLEEP")
	totalDurationStr := formatDurationFromMs(ssc.totalDuration)
	trendStr := ssc.getTrendIndicator()

	rightValue := durationStyle.Render(totalDurationStr) + trendStr

	leftWidth := lipgloss.Width(leftLabel)
	rightWidth := lipgloss.Width(rightValue)
	padding := max(width-leftWidth-rightWidth, 1)

	headerRow := leftLabel + strings.Repeat(" ", padding) + rightValue
	sections = append(sections, headerRow)

	if ssc.baselineDuration > 0 {
		sections = append(sections, ssc.renderBaselineRow(width))
	}

	return append(sections, "")
}

// getTrendIndicator returns the trend arrow based on comparison with baseline.
func (ssc *SleepStagesChart) getTrendIndicator() string {
	if ssc.baselineDuration <= 0 {
		return ""
	}
	if ssc.totalDuration > ssc.baselineDuration {
		return " " + lipgloss.NewStyle().Foreground(theme.ColorTeal).Render(theme.SymbolArrowUp)
	}
	if ssc.totalDuration < ssc.baselineDuration {
		return " " + lipgloss.NewStyle().Foreground(theme.ColorOrange).Render(theme.SymbolArrowDown)
	}
	return ""
}

// renderBaselineRow renders the baseline duration subtext row.
func (ssc *SleepStagesChart) renderBaselineRow(width int) string {
	baselineStr := formatDurationFromMs(ssc.baselineDuration)
	subValueStyle := lipgloss.NewStyle().Foreground(theme.ColorDim)
	subValueText := subValueStyle.Render(baselineStr)
	subPadding := width - lipgloss.Width(subValueText)
	return strings.Repeat(" ", max(subPadding, 0)) + subValueText
}

func (ssc *SleepStagesChart) renderStageRow(stage SleepStage, width int) string {
	var lines []string

	circleStyle := lipgloss.NewStyle().Foreground(stage.Color)
	circle := circleStyle.Render(theme.SymbolCircleEmpty)

	nameStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite).Bold(true)
	pctStyle := lipgloss.NewStyle().Foreground(stage.Color)
	durationStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite)

	stageName := nameStyle.Render(stage.Name)
	stagePct := pctStyle.Render(fmt.Sprintf("  %.0f%%", stage.Percentage))
	stageDuration := durationStyle.Render(formatDurationFromMs(stage.DurationMs))

	leftPart := circle + " " + stageName + stagePct
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(stageDuration)
	padding := max(width-leftWidth-rightWidth, 1)

	firstLine := leftPart + strings.Repeat(" ", padding) + stageDuration
	lines = append(lines, firstLine)

	barWidth := max(width-rightWidth-1, 10)

	barLine := ssc.renderStageBar(stage, barWidth)
	lines = append(lines, barLine)

	return strings.Join(lines, "\n")
}

func (ssc *SleepStagesChart) renderStageBar(stage SleepStage, barWidth int) string {
	maxPct := 100.0
	actualWidth := max(int(float64(barWidth)*(stage.Percentage/maxPct)), 0)

	var result strings.Builder

	filledStyle := lipgloss.NewStyle().Foreground(stage.Color)
	for range actualWidth {
		result.WriteString(filledStyle.Render(theme.SymbolBlockFull))
	}

	return result.String()
}

// formatDurationFromMs formats milliseconds as "H:MM".
func formatDurationFromMs(ms int) string {
	totalMinutes := ms / (1000 * 60)
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	return fmt.Sprintf("%d:%02d", hours, minutes)
}
