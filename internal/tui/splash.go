package tui

import "time"

const splashDuration = 1500 * time.Millisecond

type SplashTickMsg struct{}

type SplashState struct{}

func (m *Model) SplashView() string {
	return m.LogoView()
}
