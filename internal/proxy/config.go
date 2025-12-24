package proxy

import (
	"github.com/caarlos0/env/v11"
	appenv "github.com/garrettladley/thoop/internal/env"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/storage"
)

type Config struct {
	Port      string              `env:"PORT" envDefault:"8080"`
	Env       appenv.Environment  `env:"ENV" envDefault:"development"`
	BaseURL   string              `env:"BASE_URL,required"`
	Whoop     Whoop               `envPrefix:"WHOOP_"`
	RateLimit RateLimit           `envPrefix:"RATE_"`
	Redis     storage.RedisConfig `envPrefix:"REDIS_"`
}

type Whoop struct {
	ClientID     string `env:"CLIENT_ID,required"`
	ClientSecret string `env:"CLIENT_SECRET,required"`
}

type RateLimit struct {
	Limit float64 `env:"LIMIT" envDefault:"10"`
	Burst int     `env:"BURST" envDefault:"20"`
}

var _ oauth.ConfigProvider = (*Config)(nil)

func (c Config) GetClientID() string     { return c.Whoop.ClientID }
func (c Config) GetClientSecret() string { return c.Whoop.ClientSecret }
func (c Config) GetRedirectURL() string  { return c.BaseURL + "/auth/callback" }

func ReadConfig() (Config, error) {
	return env.ParseAs[Config]()
}
