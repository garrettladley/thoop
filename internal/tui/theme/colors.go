package theme

import "charm.land/lipgloss/v2"

var (
	ColorBlack = lipgloss.Color("#000000")
	ColorWhite = lipgloss.Color("#FFFFFF")
	ColorDim   = lipgloss.Color("#666666")
)

var (
	ColorTeal           = lipgloss.Color("#00F19F") // CTA, highlights, positive evaluations, Sleep Need
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
