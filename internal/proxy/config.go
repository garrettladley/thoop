package proxy

import "github.com/caarlos0/env/v11"

type Config struct {
	Port         string `env:"PORT" envDefault:"8080"`
	ClientID     string `env:"WHOOP_CLIENT_ID,required"`
	ClientSecret string `env:"WHOOP_CLIENT_SECRET,required"`
	BaseURL      string `env:"BASE_URL,required"`
}

func (c Config) RedirectURL() string {
	return c.BaseURL + "/auth/callback"
}

func ReadConfig() (Config, error) {
	return env.ParseAs[Config]()
}
