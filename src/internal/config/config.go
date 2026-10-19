package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	DefaultAppEnv = "development"
	defaultPort   = 8000
	minPort       = 1
	maxPort       = 65535
)

type Config struct {
	AppEnv string
	Port   int
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv: os.Getenv("APP_ENV"),
		Port:   defaultPort,
	}

	if cfg.AppEnv == "" {
		cfg.AppEnv = DefaultAppEnv
	}

	if value := os.Getenv("PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port < minPort || port > maxPort {
			return Config{}, fmt.Errorf(
				"PORT must be an integer between %d and %d",
				minPort,
				maxPort,
			)
		}

		cfg.Port = port
	}

	return cfg, nil
}
