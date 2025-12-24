package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Whoop Whoop `envPrefix:"WHOOP_"`
}

type Whoop struct {
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
	RedirectURL  string `env:"REDIRECT_URL"`
}

func Read() (Config, error) {
	return env.ParseAs[Config]()
}
