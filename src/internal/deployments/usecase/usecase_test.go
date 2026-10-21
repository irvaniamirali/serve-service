package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/deployments"
)

func TestCreateDeploymentValidatesAndNormalizesCommit(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	create := NewCreateDeployment(repository, applicationReaderFake{})

	created, err := create.Execute(
		context.Background(),
		1,
		" ABCDEF0123456789ABCDEF0123456789ABCDEF01 ",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if created.Status != deployments.StatusQueued ||
		repository.commitSHA == nil ||
		*repository.commitSHA != "abcdef0123456789abcdef0123456789abcdef01" {
		t.Fatalf("created deployment = %+v, commit SHA = %v", created, repository.commitSHA)
	}

	_, err = create.Execute(context.Background(), 1, "not-a-commit")
	if !errors.Is(err, deployments.ErrValidation) {
		t.Fatalf("invalid commit error = %v, want ErrValidation", err)
	}
	if repository.createCalls != 1 {
		t.Fatalf("Create calls = %d, want 1", repository.createCalls)
	}
}

func TestCreateDeploymentRequiresExistingApplication(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	create := NewCreateDeployment(repository, applicationReaderFake{missing: true})

	_, err := create.Execute(context.Background(), 1, "")
	if !errors.Is(err, applications.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want applications.ErrNotFound", err)
	}
	if repository.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", repository.createCalls)
	}
}

func TestTransitionDeploymentStatusValidatesAndStoresFailureMessage(t *testing.T) {
	repository := &deploymentRepositoryFake{
		deployment: deployments.Deployment{
			ID:     1,
			Status: deployments.StatusQueued,
		},
	}
	transition := NewTransitionDeploymentStatus(repository)

	if _, err := transition.Execute(context.Background(), 1, deployments.StatusFailed, ""); !errors.Is(err, deployments.ErrValidation) {
		t.Fatalf("failure without diagnostic error = %v, want ErrValidation", err)
	}
	if _, err := transition.Execute(context.Background(), 1, deployments.StatusBuilding, "not allowed"); !errors.Is(err, deployments.ErrValidation) {
		t.Fatalf("diagnostic on non-failure error = %v, want ErrValidation", err)
	}

	if _, err := transition.Execute(context.Background(), 1, deployments.StatusBuilding, ""); err != nil {
		t.Fatalf("transition to building: %v", err)
	}
	failed, err := transition.Execute(
		context.Background(),
		1,
		deployments.StatusFailed,
		" build failed ",
	)
	if err != nil {
		t.Fatalf("transition to failed: %v", err)
	}
	if failed.Status != deployments.StatusFailed ||
		failed.ErrorMessage == nil ||
		*failed.ErrorMessage != "build failed" {
		t.Fatalf("failure transition = %+v", failed)
	}
}

type applicationReaderFake struct {
	missing bool
}

func (reader applicationReaderFake) GetApplicationByID(
	_ context.Context,
	id int64,
) (applications.Application, error) {
	if reader.missing || id != 1 {
		return applications.Application{}, applications.ErrNotFound
	}
	return applications.Application{ID: id}, nil
}

type deploymentRepositoryFake struct {
	deployment  deployments.Deployment
	commitSHA   *string
	createCalls int
}

func (repository *deploymentRepositoryFake) Create(
	_ context.Context,
	applicationID int64,
	commitSHA *string,
) (deployments.Deployment, error) {
	repository.createCalls++
	repository.commitSHA = commitSHA
	repository.deployment = deployments.Deployment{
		ID:            1,
		ApplicationID: applicationID,
		Status:        deployments.StatusQueued,
		CommitSHA:     commitSHA,
	}
	return repository.deployment, nil
}

func (repository *deploymentRepositoryFake) GetByID(
	_ context.Context,
	id int64,
) (deployments.Deployment, error) {
	if repository.deployment.ID != id {
		return deployments.Deployment{}, deployments.ErrNotFound
	}
	return repository.deployment, nil
}

func (*deploymentRepositoryFake) ListByApplication(
	context.Context,
	int64,
	int,
	int,
) ([]deployments.Deployment, error) {
	return []deployments.Deployment{}, nil
}

func (repository *deploymentRepositoryFake) TransitionStatus(
	_ context.Context,
	id int64,
	expectedStatus string,
	nextStatus string,
	failureMessage *string,
) (deployments.Deployment, error) {
	if repository.deployment.ID != id {
		return deployments.Deployment{}, deployments.ErrNotFound
	}
	if repository.deployment.Status != expectedStatus {
		return deployments.Deployment{}, deployments.ErrConcurrentTransition
	}
	repository.deployment.Status = nextStatus
	repository.deployment.ErrorMessage = failureMessage
	return repository.deployment, nil
}
