package tui

import "time"

const splashDuration = 1500 * time.Millisecond

type SplashTickMsg struct{}

type AuthStatusMsg struct {
	HasToken bool
	Err      error
}

type MetricsDataMsg struct {
	Sleep    *float64 // SleepPerformancePercentage (0-100)
	Recovery *float64 // RecoveryScore (0-100)
	Strain   *float64 // Strain (0-21)
	Err      error
}
