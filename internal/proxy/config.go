package proxy

import (
	"github.com/caarlos0/env/v11"
	appenv "github.com/garrettladley/thoop/internal/env"
	"github.com/garrettladley/thoop/internal/oauth"
)

type Config struct {
	Port           string             `env:"PORT" envDefault:"8080"`
	Env            appenv.Environment `env:"ENV" envDefault:"development"`
	BaseURL        string             `env:"BASE_URL,required"`
	Whoop          Whoop              `envPrefix:"WHOOP_"`
	RateLimit      RateLimit          `envPrefix:"RATE_"`
	WhoopRateLimit WhoopRateLimit     `envPrefix:"WHOOP_RATE_LIMIT_"`
	Redis          Redis              `envPrefix:"REDIS_"`
}

type Redis struct {
	URL string `env:"URL"`
}

type Whoop struct {
	ClientID     string `env:"CLIENT_ID,required"`
	ClientSecret string `env:"CLIENT_SECRET,required"`
}

type RateLimit struct {
	Limit float64 `env:"LIMIT" envDefault:"10"`
	Burst int     `env:"BURST" envDefault:"20"`
}

type WhoopRateLimit struct {
	// max 10 users as unapproved app, assumes ~5 concurrent
	PerUserMinuteLimit int `env:"PER_USER_MINUTE_LIMIT" envDefault:"20"`
	PerUserDayLimit    int `env:"PER_USER_DAY_LIMIT" envDefault:"2000"`

	// w/ safety buffers from WHOOP's 100/min, 10k/day
	GlobalMinuteLimit int `env:"GLOBAL_MINUTE_LIMIT" envDefault:"95"`
	GlobalDayLimit    int `env:"GLOBAL_DAY_LIMIT" envDefault:"9950"`
}

var _ oauth.ConfigProvider = (*Config)(nil)

func (c Config) GetClientID() string     { return c.Whoop.ClientID }
func (c Config) GetClientSecret() string { return c.Whoop.ClientSecret }
func (c Config) GetRedirectURL() string  { return c.BaseURL + "/auth/callback" }

func ReadConfig() (Config, error) {
	return env.ParseAs[Config]()
}
