package config

import "github.com/caarlos0/env/v11"

type Config struct {
	ProxyURL string `env:"PROXY_URL,required"`
	Whoop    Whoop  `envPrefix:"WHOOP_"`
}

type Whoop struct {
	ClientID     string `env:"CLIENT_ID,required"`
	ClientSecret string `env:"CLIENT_SECRET,required"`
	RedirectURL  string `env:"REDIRECT_URL,required"`
}

func Read() (Config, error) {
	return env.ParseAs[Config]()
}
