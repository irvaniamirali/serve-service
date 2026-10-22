package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
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
		databaseCommitResolverFake{},
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
	if failedDeployment.CommitSHA == nil || *failedDeployment.CommitSHA != "abcdef0123456789abcdef0123456789abcdef01" {
		t.Fatalf("deployment default-branch SHA = %v, want persisted resolved SHA", failedDeployment.CommitSHA)
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

func TestPostgresBuildQueueClaiming(t *testing.T) {
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

	var pendingBuilds int64
	if err := db.WithContext(ctx).
		Table("deployments").
		Where("status IN ?", []string{deploymentdomain.StatusQueued, deploymentdomain.StatusBuilding}).
		Count(&pendingBuilds).Error; err != nil {
		t.Fatalf("check integration database queue: %v", err)
	}
	if pendingBuilds != 0 {
		t.Skip("PostgreSQL build queue integration test requires no pre-existing queued or building deployments")
	}

	applicationRepository := applicationpostgres.NewRepository(db)
	application, err := applicationRepository.CreateApplication(ctx, applicationdomain.Application{
		Name:          fmt.Sprintf("build-queue-integration-%d", time.Now().UnixNano()),
		RepositoryURL: "https://github.com/acme/build-queue-integration.git",
		Branch:        "main",
	})
	if err != nil {
		t.Fatalf("create integration application: %v", err)
	}
	deploymentRepository := deploymentpostgres.NewRepository(db)
	t.Cleanup(func() {
		_ = db.WithContext(context.Background()).Exec(
			"DELETE FROM deployments WHERE application_id = ?",
			application.ID,
		).Error
		_ = db.WithContext(context.Background()).Exec(
			"DELETE FROM applications WHERE id = ?",
			application.ID,
		).Error
	})

	if _, err := deploymentRepository.ClaimNextQueued(ctx); !errors.Is(err, deploymentdomain.ErrNoQueuedBuild) {
		t.Fatalf("empty queue claim error = %v, want ErrNoQueuedBuild", err)
	}

	commitSHA := "abcdef0123456789abcdef0123456789abcdef01"
	first, err := deploymentRepository.Create(ctx, application.ID, &commitSHA)
	if err != nil {
		t.Fatalf("create first queued deployment: %v", err)
	}
	second, err := deploymentRepository.Create(ctx, application.ID, &commitSHA)
	if err != nil {
		t.Fatalf("create second queued deployment: %v", err)
	}

	type claimResponse struct {
		deployment deploymentdomain.Deployment
		err        error
	}
	start := make(chan struct{})
	responses := make(chan claimResponse, 2)
	var claimers sync.WaitGroup
	for range 2 {
		claimers.Add(1)
		go func() {
			defer claimers.Done()
			<-start
			claimed, claimErr := deploymentRepository.ClaimNextQueued(ctx)
			responses <- claimResponse{deployment: claimed, err: claimErr}
		}()
	}
	close(start)
	claimers.Wait()
	close(responses)

	var claimed deploymentdomain.Deployment
	emptyResults := 0
	for response := range responses {
		if errors.Is(response.err, deploymentdomain.ErrNoQueuedBuild) {
			emptyResults++
			continue
		}
		if response.err != nil {
			t.Fatalf("concurrent queue claim failed: %v", response.err)
		}
		if response.deployment.Status != deploymentdomain.StatusBuilding {
			t.Fatalf("claimed status = %q, want building", response.deployment.Status)
		}
		if response.deployment.ID != first.ID && response.deployment.ID != second.ID {
			t.Fatalf("claimed unrelated deployment %d", response.deployment.ID)
		}
		claimed = response.deployment
	}
	if claimed.ID == 0 || emptyResults != 1 {
		t.Fatalf("concurrent claims: claimed ID=%d, empty results=%d, want one claim and one empty result", claimed.ID, emptyResults)
	}

	persistedClaim, err := deploymentRepository.GetByID(ctx, claimed.ID)
	if err != nil {
		t.Fatalf("read claimed deployment: %v", err)
	}
	if persistedClaim.Status != deploymentdomain.StatusBuilding || persistedClaim.BuildStartedAt == nil {
		t.Fatalf("claimed deployment state = %q, started at %v; want building with start time", persistedClaim.Status, persistedClaim.BuildStartedAt)
	}

	imageReference := "integration/image:" + commitSHA
	imageID := "sha256:" + strings.Repeat("a", 64)
	built, err := deploymentRepository.RecordBuildResult(
		ctx,
		claimed.ID,
		true,
		&imageReference,
		&imageID,
		"integration build succeeded",
		nil,
	)
	if err != nil {
		t.Fatalf("record successful build: %v", err)
	}
	if built.Status != deploymentdomain.StatusBuilt || built.BuildFinishedAt == nil {
		t.Fatalf("successful build result = status %q, finished at %v", built.Status, built.BuildFinishedAt)
	}

	remainingID := first.ID
	if claimed.ID == first.ID {
		remainingID = second.ID
	}
	next, err := deploymentRepository.ClaimNextQueued(ctx)
	if err != nil {
		t.Fatalf("claim remaining deployment: %v", err)
	}
	if next.ID != remainingID || next.Status != deploymentdomain.StatusBuilding {
		t.Fatalf("next claim = %d/%q, want %d/building", next.ID, next.Status, remainingID)
	}
	failureMessage := "integration build failure"
	failed, err := deploymentRepository.RecordBuildResult(
		ctx,
		next.ID,
		false,
		nil,
		nil,
		"integration build failed",
		&failureMessage,
	)
	if err != nil {
		t.Fatalf("record failed build: %v", err)
	}
	if failed.Status != deploymentdomain.StatusFailed || failed.ErrorMessage == nil || *failed.ErrorMessage != failureMessage {
		t.Fatalf("failed build result = status %q, error %v", failed.Status, failed.ErrorMessage)
	}
	if _, err := deploymentRepository.ClaimNextQueued(ctx); !errors.Is(err, deploymentdomain.ErrNoQueuedBuild) {
		t.Fatalf("drained queue claim error = %v, want ErrNoQueuedBuild", err)
	}

	closedDB, err := database.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open database for failure propagation check: %v", err)
	}
	closedSQLDB, err := closedDB.DB()
	if err != nil {
		t.Fatalf("get failure-check SQL database: %v", err)
	}
	if err := closedSQLDB.Close(); err != nil {
		t.Fatalf("close failure-check database: %v", err)
	}
	_, err = deploymentpostgres.NewRepository(closedDB).ClaimNextQueued(ctx)
	if err == nil || errors.Is(err, deploymentdomain.ErrNoQueuedBuild) {
		t.Fatalf("closed database claim error = %v, want a propagated database error", err)
	}
}

type databaseCommitResolverFake struct{}

func (databaseCommitResolverFake) ResolveDefaultBranchHead(
	context.Context,
	applicationdomain.Application,
) (string, error) {
	return "abcdef0123456789abcdef0123456789abcdef01", nil
}
