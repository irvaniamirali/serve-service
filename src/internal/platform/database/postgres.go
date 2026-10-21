package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/irvaniamirali/serve-service/internal/platform/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const connectionTimeout = 5 * time.Second

// Open creates and verifies the shared GORM connection to PostgreSQL.
func Open(ctx context.Context, cfg config.Config) (*gorm.DB, error) {
	databaseURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, strconv.Itoa(cfg.DBPort)),
		Path:   cfg.DBName,
	}

	query := databaseURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	databaseURL.RawQuery = query.Encode()

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			DSN: databaseURL.String(),
		}),
		&gorm.Config{
			// Verify connectivity explicitly using the startup context.
			DisableAutomaticPing: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("open GORM database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying SQL database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		if closeErr := sqlDB.Close(); closeErr != nil {
			return nil, fmt.Errorf(
				"ping PostgreSQL: %w (close database: %v)",
				err,
				closeErr,
			)
		}

		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return db, nil
}
