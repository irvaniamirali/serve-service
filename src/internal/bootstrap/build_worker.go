package bootstrap

import (
	"context"
	"log/slog"

	applicationpostgres "github.com/irvaniamirali/serve-service/internal/applications/adapters/postgres"
	localbuilder "github.com/irvaniamirali/serve-service/internal/builds/adapters/local"
	buildusecase "github.com/irvaniamirali/serve-service/internal/builds/usecase"
	deploymentpostgres "github.com/irvaniamirali/serve-service/internal/deployments/adapters/postgres"
	"gorm.io/gorm"
)

// StartBuildWorker runs the durable deployment builder until its context is canceled.
func StartBuildWorker(ctx context.Context, db *gorm.DB, logger *slog.Logger) <-chan error {
	worker := buildusecase.NewWorker(
		deploymentpostgres.NewRepository(db),
		applicationpostgres.NewRepository(db),
		localbuilder.NewBuilder(),
		logger,
	)

	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- worker.Run(ctx)
	}()

	return done
}
