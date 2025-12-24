package tui

const Logo = `
 ▄▄▄▄▄▄▄▄  ▄▄    ▄▄    ▄▄▄▄      ▄▄▄▄    ▄▄▄▄▄▄
 ▀▀▀██▀▀▀  ██    ██   ██▀▀██    ██▀▀██   ██▀▀▀▀█▄
    ██     ██    ██  ██    ██  ██    ██  ██    ██
    ██     ████████  ██    ██  ██    ██  ██████▀
    ██     ██    ██  ██    ██  ██    ██  ██
    ██     ██    ██   ██▄▄██    ██▄▄██   ██
    ▀▀     ▀▀    ▀▀    ▀▀▀▀      ▀▀▀▀    ▀▀`

func (m *Model) LogoView() string {
	return m.theme.TextAccent().Render(Logo)
}
