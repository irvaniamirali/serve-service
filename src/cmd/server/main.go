package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/irvaniamirali/serve-service/internal/config"
	httpserver "github.com/irvaniamirali/serve-service/internal/http"
	"github.com/irvaniamirali/serve-service/internal/logging"
	"github.com/joho/godotenv"
)

const (
	shutdownTimeout = 10 * time.Second
	envFilePath     = ".env"
)

func main() {
	logger := logging.New()

	if err := run(logger); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	if err := loadDevelopmentEnv(); err != nil {
		return fmt.Errorf("load development environment: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	app := httpserver.NewServer()
	address := fmt.Sprintf(":%d", cfg.Port)

	listenErrors := make(chan error, 1)

	go func() {
		listenErrors <- app.Listen(address)
	}()

	logger.Info(
		"server started",
		"address", address,
		"environment", cfg.AppEnv,
	)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-listenErrors:
		if err != nil {
			return fmt.Errorf("listen on %s: %w", address, err)
		}

		return nil

	case sig := <-signals:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := <-listenErrors; err != nil {
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	}

	logger.Info("server stopped")

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
