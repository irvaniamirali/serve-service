package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/irvaniamirali/serve-service/internal/bootstrap"
	"github.com/irvaniamirali/serve-service/internal/infrastructure/config"
	"github.com/irvaniamirali/serve-service/internal/infrastructure/database"
	"github.com/irvaniamirali/serve-service/internal/infrastructure/logging"
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

	db, err := database.Open(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying SQL database: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	workerContext, stopWorker := context.WithCancel(context.Background())
	workerErrors := bootstrap.StartBuildWorker(workerContext, db, logger)
	defer func() {
		stopWorker()
		<-workerErrors
	}()

	app := bootstrap.NewServer(db)

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
		stopWorker()
		<-workerErrors
		if err != nil {
			return fmt.Errorf("listen on %s: %w", address, err)
		}

		return nil

	case err := <-workerErrors:
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		shutdownErr := app.ShutdownWithContext(shutdownContext)
		listenErr := <-listenErrors
		if shutdownErr != nil {
			return fmt.Errorf("build worker stopped unexpectedly: %v; shutdown server: %w", err, shutdownErr)
		}
		if listenErr != nil {
			return fmt.Errorf("build worker stopped unexpectedly: %v; server stopped: %w", err, listenErr)
		}
		if err == nil {
			return fmt.Errorf("build worker stopped unexpectedly")
		}
		return fmt.Errorf("build worker stopped unexpectedly: %w", err)

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

	stopWorker()
	if err := <-workerErrors; err != nil {
		return fmt.Errorf("stop build worker: %w", err)
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
