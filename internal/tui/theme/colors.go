package theme

import "charm.land/lipgloss/v2"

var (
	ColorBlack   = lipgloss.Color("#000000")
	ColorWhite   = lipgloss.Color("#FFFFFF")
	ColorDim     = lipgloss.Color("#666666")
	ColorNeutral = lipgloss.Color("#8D8E92") // No change indicator
	ColorNavHint = lipgloss.Color("#3a3a3a")
)

var (
	ColorTeal           = lipgloss.Color("#00F19F") // CTA, highlights, positive evaluations, Sleep Need
	ColorOrange         = lipgloss.Color("#F1AE45") // Negative trend indicator
	ColorStrain         = lipgloss.Color("#0093E7") // Activities and Strain related
	ColorRecoveryBlue   = lipgloss.Color("#67AEE6") // Recovery data without valuation
	ColorHighRecovery   = lipgloss.Color("#16EC06") // Recovery 100-67%
	ColorMediumRecovery = lipgloss.Color("#FFDE00") // Recovery 66-34%
	ColorLowRecovery    = lipgloss.Color("#FF0026") // Recovery 33-0%
	ColorSleep          = lipgloss.Color("#7BA1BB") // Sleep related data
)

var (
	ColorBgDark  = lipgloss.Color("#101518") // Darker end of gradient
	ColorBgLight = lipgloss.Color("#283339") // Lighter end of gradient
)

var (
	ColorHRZone1 = lipgloss.Color("#4A90D9") // Zone 1 (lowest)
	ColorHRZone2 = lipgloss.Color("#2ECC71") // Zone 2
	ColorHRZone3 = lipgloss.Color("#F39C12") // Zone 3 (highest)
)

var (
	ColorSleepAwake = lipgloss.Color("#E74C3C") // Awake
	ColorSleepREM   = lipgloss.Color("#3498DB") // REM sleep
	ColorSleepLight = lipgloss.Color("#2ECC71") // Light sleep
	ColorSleepDeep  = lipgloss.Color("#9B59B6") // Deep sleep
)

var (
	ColorChartGrid = lipgloss.Color("#333333") // Grid lines
)
