package proxy

import (
	"github.com/caarlos0/env/v11"
	"github.com/garrettladley/thoop/internal/oauth"
)

type Config struct {
	Port         string  `env:"PORT" envDefault:"8080"`
	ClientID     string  `env:"WHOOP_CLIENT_ID,required"`
	ClientSecret string  `env:"WHOOP_CLIENT_SECRET,required"`
	BaseURL      string  `env:"BASE_URL,required"`
	RateLimit    float64 `env:"RATE_LIMIT" envDefault:"10"`
	RateBurst    int     `env:"RATE_BURST" envDefault:"20"`
}

var _ oauth.ConfigProvider = Config{}

func (c Config) GetClientID() string     { return c.ClientID }
func (c Config) GetClientSecret() string { return c.ClientSecret }
func (c Config) GetRedirectURL() string  { return c.BaseURL + "/auth/callback" }

func ReadConfig() (Config, error) {
	return env.ParseAs[Config]()
}
