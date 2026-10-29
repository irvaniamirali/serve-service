package bootstrap

import (
	"context"
	"log/slog"

	"github.com/irvaniamirali/serve-service/internal/applications"
	buildusecase "github.com/irvaniamirali/serve-service/internal/builds"
	localbuilder "github.com/irvaniamirali/serve-service/internal/builds/local"
	"github.com/irvaniamirali/serve-service/internal/deployments"
	"gorm.io/gorm"
)

// StartBuildWorker runs the durable deployment builder until its context is canceled.
func StartBuildWorker(ctx context.Context, db *gorm.DB, logger *slog.Logger) <-chan error {
	worker := buildusecase.NewWorker(
		deployments.NewDeploymentRepository(db),
		applications.NewApplicationRepository(db),
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
