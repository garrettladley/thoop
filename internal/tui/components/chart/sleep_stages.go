package chart

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

// Sleep stage colors
var (
	ColorSleepLight = lipgloss.Color("#A4A3EB")
	ColorSleepAwake = lipgloss.Color("#C8C8C8")
	ColorSleepDeep  = lipgloss.Color("#EC9AF4")
	ColorSleepREM   = lipgloss.Color("#A15EE5")
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
	stages           []SleepStage
	totalDuration    int // total duration in milliseconds
	showHeader       bool
	baselineDuration int // baseline duration in milliseconds for trend comparison
}

// SleepStagesChartOption configures a SleepStagesChart.
type SleepStagesChartOption func(*SleepStagesChart)

// WithSleepStagesShowHeader shows or hides the header.
func WithSleepStagesShowHeader(show bool) SleepStagesChartOption {
	return func(ssc *SleepStagesChart) {
		ssc.showHeader = show
	}
}

// WithSleepStagesBaseline sets the baseline duration for trend comparison.
func WithSleepStagesBaseline(baselineMs int) SleepStagesChartOption {
	return func(ssc *SleepStagesChart) {
		ssc.baselineDuration = baselineMs
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

	// pre-allocate: header (2 lines) + stages (2 lines each)
	sections := make([]string, 0, 2+len(ssc.stages)*2)

	// header row: "HOURS OF SLEEP" on left, duration with trend on right
	if ssc.showHeader {
		labelStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite).Bold(true)
		durationStyle := lipgloss.NewStyle().Foreground(theme.ColorWhite).Bold(true)

		leftLabel := labelStyle.Render("HOURS OF SLEEP")
		totalDurationStr := formatDurationFromMs(ssc.totalDuration)

		// determine trend direction
		var trendStr string
		if ssc.baselineDuration > 0 {
			if ssc.totalDuration > ssc.baselineDuration {
				trendStr = " " + lipgloss.NewStyle().Foreground(theme.ColorTeal).Render("▲")
			} else if ssc.totalDuration < ssc.baselineDuration {
				trendStr = " " + lipgloss.NewStyle().Foreground(theme.ColorOrange).Render("▼")
			}
		}

		rightValue := durationStyle.Render(totalDurationStr) + trendStr

		leftWidth := lipgloss.Width(leftLabel)
		rightWidth := lipgloss.Width(rightValue)
		padding := max(width-leftWidth-rightWidth, 1)

		headerRow := leftLabel + strings.Repeat(" ", padding) + rightValue
		sections = append(sections, headerRow)

		// add baseline subtext row if available
		if ssc.baselineDuration > 0 {
			baselineStr := formatDurationFromMs(ssc.baselineDuration)
			subValueStyle := lipgloss.NewStyle().Foreground(theme.ColorDim)
			subValueText := subValueStyle.Render(baselineStr)
			subPadding := width - lipgloss.Width(subValueText)
			subRow := strings.Repeat(" ", max(subPadding, 0)) + subValueText
			sections = append(sections, subRow)
		}

		sections = append(sections, "")
	}

	// render each stage
	for _, stage := range ssc.stages {
		stageRow := ssc.renderStageRow(stage, width)
		sections = append(sections, stageRow)
	}

	return strings.Join(sections, "\n")
}

func (ssc *SleepStagesChart) renderStageRow(stage SleepStage, width int) string {
	var lines []string

	circleStyle := lipgloss.NewStyle().Foreground(stage.Color)
	circle := circleStyle.Render("○")

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
		result.WriteString(filledStyle.Render("█"))
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
