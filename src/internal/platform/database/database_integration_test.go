package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	applicationdomain "github.com/irvaniamirali/serve-service/internal/applications"
	applicationpostgres "github.com/irvaniamirali/serve-service/internal/applications/adapters/postgres"
	"github.com/irvaniamirali/serve-service/internal/applications/usecase"
	deploymentdomain "github.com/irvaniamirali/serve-service/internal/deployments"
	deploymentpostgres "github.com/irvaniamirali/serve-service/internal/deployments/adapters/postgres"
	deploymentusecase "github.com/irvaniamirali/serve-service/internal/deployments/usecase"
	"github.com/irvaniamirali/serve-service/internal/platform/config"
	"github.com/irvaniamirali/serve-service/internal/platform/database"
)

func TestPostgresPersistenceAndMigrations(t *testing.T) {
	if os.Getenv("RUN_DATABASE_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_DATABASE_INTEGRATION_TESTS=1 to run PostgreSQL integration checks")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load database configuration: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.RunMigrations(ctx, cfg); err != nil {
		t.Fatalf("apply database migrations: %v", err)
	}
	db, err := database.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open GORM database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying SQL database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	applicationRepository := applicationpostgres.NewRepository(db)
	createApplication := usecase.NewCreateApplication(applicationRepository)
	deploymentRepository := deploymentpostgres.NewRepository(db)
	createDeployment := deploymentusecase.NewCreateDeployment(
		deploymentRepository,
		applicationRepository,
	)
	listDeploymentsByApplication := deploymentusecase.NewListDeploymentsByApplication(
		deploymentRepository,
		applicationRepository,
	)
	transitionDeploymentStatus := deploymentusecase.NewTransitionDeploymentStatus(
		deploymentRepository,
	)

	application, err := createApplication.Execute(
		ctx,
		usecase.CreateApplicationInput{
			Name:          fmt.Sprintf("m02-integration-%d", time.Now().UnixNano()),
			RepositoryURL: "https://github.com/acme/integration.git",
			Branch:        "main",
		},
	)
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	if _, err := createApplication.Execute(
		ctx,
		usecase.CreateApplicationInput{
			Name:          application.Name,
			RepositoryURL: "https://github.com/acme/duplicate.git",
			Branch:        "main",
		},
	); !errors.Is(err, applicationdomain.ErrConflict) {
		t.Fatalf("duplicate application creation error = %v, want ErrConflict", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec(
			"DELETE FROM deployments WHERE application_id = ?",
			application.ID,
		)
		db.WithContext(context.Background()).Exec(
			"DELETE FROM applications WHERE id = ?",
			application.ID,
		)
	})

	deployment, err := createDeployment.Execute(
		ctx,
		application.ID,
		"abcdef0123456789abcdef0123456789abcdef01",
	)
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	if deployment.Status != deploymentdomain.StatusQueued {
		t.Fatalf("deployment status = %q, want %q", deployment.Status, deploymentdomain.StatusQueued)
	}
	history, err := listDeploymentsByApplication.Execute(ctx, application.ID, 10, 0)
	if err != nil {
		t.Fatalf("list deployment history: %v", err)
	}
	if len(history) != 1 || history[0].ID != deployment.ID {
		t.Fatalf("deployment history = %+v, want deployment %d", history, deployment.ID)
	}

	transitionResults := make(chan error, 2)
	var transitions sync.WaitGroup
	for _, target := range []string{deploymentdomain.StatusBuilding, deploymentdomain.StatusCancelled} {
		transitions.Add(1)
		go func(status string) {
			defer transitions.Done()
			_, transitionErr := deploymentRepository.TransitionStatus(
				ctx,
				deployment.ID,
				deploymentdomain.StatusQueued,
				status,
				nil,
			)
			transitionResults <- transitionErr
		}(target)
	}
	transitions.Wait()
	close(transitionResults)
	successes := 0
	conflicts := 0
	for transitionErr := range transitionResults {
		if transitionErr == nil {
			successes++
		} else {
			conflicts++
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent transitions: successes=%d conflicts=%d, want one each", successes, conflicts)
	}

	failedDeployment, err := createDeployment.Execute(ctx, application.ID, "")
	if err != nil {
		t.Fatalf("create deployment without known commit: %v", err)
	}
	if _, err := transitionDeploymentStatus.Execute(ctx, failedDeployment.ID, deploymentdomain.StatusBuilding, ""); err != nil {
		t.Fatalf("transition deployment to building: %v", err)
	}
	failedDeployment, err = transitionDeploymentStatus.Execute(
		ctx,
		failedDeployment.ID,
		deploymentdomain.StatusFailed,
		"integration failure diagnostic",
	)
	if err != nil {
		t.Fatalf("transition deployment to failed: %v", err)
	}
	if failedDeployment.ErrorMessage == nil || *failedDeployment.ErrorMessage != "integration failure diagnostic" {
		t.Fatalf("failure diagnostic = %v, want persisted diagnostic", failedDeployment.ErrorMessage)
	}

	history, err = listDeploymentsByApplication.Execute(ctx, application.ID, 10, 0)
	if err != nil {
		t.Fatalf("list deployment history after second deployment: %v", err)
	}
	if len(history) != 2 || history[0].ID != failedDeployment.ID {
		t.Fatalf("deployment history ordering = %+v, want newest deployment first", history)
	}

	persistedDB, err := database.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("reopen GORM database: %v", err)
	}
	persistedSQLDB, err := persistedDB.DB()
	if err != nil {
		t.Fatalf("get reopened SQL database: %v", err)
	}
	t.Cleanup(func() { _ = persistedSQLDB.Close() })
	persistedApplicationRepository := applicationpostgres.NewRepository(persistedDB)
	persistedGetApplicationByID := usecase.NewGetApplicationByID(persistedApplicationRepository)
	persistedDeploymentRepository := deploymentpostgres.NewRepository(persistedDB)
	persistedApplication, err := persistedGetApplicationByID.Execute(ctx, application.ID)
	if err != nil {
		t.Fatalf("retrieve application after reopening pool: %v", err)
	}
	if persistedApplication.Branch != "main" {
		t.Errorf("persisted branch = %q, want main", persistedApplication.Branch)
	}
	persistedDeployment, err := persistedDeploymentRepository.GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatalf("retrieve deployment after reopening pool: %v", err)
	}
	if persistedDeployment.Status != deploymentdomain.StatusBuilding && persistedDeployment.Status != deploymentdomain.StatusCancelled {
		t.Errorf("persisted status = %q, want building or cancelled", persistedDeployment.Status)
	}
	persistedFailure, err := persistedDeploymentRepository.GetByID(ctx, failedDeployment.ID)
	if err != nil {
		t.Fatalf("retrieve failed deployment after reopening pool: %v", err)
	}
	if persistedFailure.ErrorMessage == nil || *persistedFailure.ErrorMessage != "integration failure diagnostic" {
		t.Errorf("persisted failure message = %v, want integration failure diagnostic", persistedFailure.ErrorMessage)
	}
}
