package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/garrettladley/thoop/internal/oauth"
)

type Config struct {
	ProxyURL string `env:"PROXY_URL,required"`
	Whoop    Whoop  `envPrefix:"WHOOP_"`
}

type Whoop struct {
	ClientID     string `env:"CLIENT_ID,required"`
	ClientSecret string `env:"CLIENT_SECRET,required"`
	RedirectURL  string `env:"REDIRECT_URL,required"`
}

var _ oauth.ConfigProvider = Whoop{}

func (w Whoop) GetClientID() string     { return w.ClientID }
func (w Whoop) GetClientSecret() string { return w.ClientSecret }
func (w Whoop) GetRedirectURL() string  { return w.RedirectURL }

func Read() (Config, error) {
	return env.ParseAs[Config]()
}
