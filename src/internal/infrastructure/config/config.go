package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	// DefaultAppEnv is used when APP_ENV is not set.
	DefaultAppEnv = "development"

	defaultPort       = 8000
	minPort           = 1
	maxPort           = 65535
	defaultDBHost     = "localhost"
	defaultDBPort     = 5432
	defaultDBUser     = "serve_service"
	defaultDBPassword = "serve_service_dev"
	defaultDBName     = "serve_service"
	defaultDBSSLMode  = "disable"
)

// Config contains validated server and database settings.
type Config struct {
	AppEnv string
	Port   int

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	GitHubWebhookSecret string
}

// Load reads and validates runtime configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:              os.Getenv("APP_ENV"),
		Port:                defaultPort,
		DBHost:              getEnv("DB_HOST", defaultDBHost),
		DBPort:              defaultDBPort,
		DBUser:              getEnv("DB_USER", defaultDBUser),
		DBPassword:          getEnv("DB_PASSWORD", defaultDBPassword),
		DBName:              getEnv("DB_NAME", defaultDBName),
		DBSSLMode:           getEnv("DB_SSLMODE", defaultDBSSLMode),
		GitHubWebhookSecret: getEnv("GITHUB_WEBHOOK_SECRET", ""),
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

	if value := os.Getenv("DB_PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port < minPort || port > maxPort {
			return Config{}, fmt.Errorf(
				"DB_PORT must be an integer between %d and %d",
				minPort,
				maxPort,
			)
		}

		cfg.DBPort = port
	}

	if cfg.DBHost == "" {
		return Config{}, fmt.Errorf("DB_HOST must not be empty")
	}
	if cfg.DBUser == "" {
		return Config{}, fmt.Errorf("DB_USER must not be empty")
	}
	if cfg.DBName == "" {
		return Config{}, fmt.Errorf("DB_NAME must not be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
