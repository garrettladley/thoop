package config

import (
	"github.com/caarlos0/env/v11"
)

const DefaultServerURL = "https://thoop.fly.dev"

type Config struct {
	ServerURL string `env:"SERVER_URL" envDefault:"https://thoop.fly.dev"`
}

func Read() (Config, error) {
	return env.ParseAs[Config]()
}
