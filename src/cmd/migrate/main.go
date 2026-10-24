package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/irvaniamirali/serve-service/internal/infrastructure/config"
	"github.com/irvaniamirali/serve-service/internal/infrastructure/database"
	"github.com/joho/godotenv"
)

const envFilePath = ".env"

func main() {
	if err := run(); err != nil {
		slog.Error("migration command failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if err := loadDevelopmentEnv(); err != nil {
		return fmt.Errorf("load development environment: %w", err)
	}

	if len(os.Args) != 2 {
		return fmt.Errorf("usage: go run ./cmd/migrate <up|down>")
	}

	command := os.Args[1]
	if command != "up" && command != "down" {
		return fmt.Errorf("unsupported migration command %q", command)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	if err := database.RunMigrationCommand(
		context.Background(),
		cfg,
		command,
	); err != nil {
		return err
	}

	slog.Info("database migrations completed", "command", command)

	return nil
}

func loadDevelopmentEnv() error {
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "" && appEnv != config.DefaultAppEnv {
		return nil
	}

	if err := godotenv.Load(envFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load %q: %w", envFilePath, err)
	}

	return nil
}
