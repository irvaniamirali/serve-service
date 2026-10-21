package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/irvaniamirali/serve-service/internal/platform/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const (
	migrationDirectory = "migrations"
	databaseDriver     = "pgx"
)

// RunMigrations applies all pending embedded database migrations.
func RunMigrations(ctx context.Context, cfg config.Config) error {
	return RunMigrationCommand(ctx, cfg, "up")
}

// RunMigrationCommand applies or rolls back migrations using command.
func RunMigrationCommand(
	ctx context.Context,
	cfg config.Config,
	command string,
) (returnErr error) {
	databaseURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, strconv.Itoa(cfg.DBPort)),
		Path:   cfg.DBName,
	}

	query := databaseURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	databaseURL.RawQuery = query.Encode()

	db, err := sql.Open(databaseDriver, databaseURL.String())
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close database for migrations: %w", err))
		}
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database for migrations: %w", err)
	}

	goose.SetBaseFS(migrationFiles)

	if err := goose.SetDialect(databaseDriver); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	switch command {
	case "up":
		err = goose.UpContext(ctx, db, migrationDirectory)
	case "down":
		err = goose.DownContext(ctx, db, migrationDirectory)
	default:
		return fmt.Errorf("unsupported migration command %q", command)
	}

	if err != nil {
		return fmt.Errorf("%s database migrations: %w", command, err)
	}

	return nil
}
