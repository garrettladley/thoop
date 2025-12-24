package tui

type SplashState struct{}

func (m *Model) SplashView() string {
	return m.LogoView()
}
