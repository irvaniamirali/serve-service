package deployments

import (
	"context"
	"errors"
	"testing"
)

func TestTransitionDeploymentStatusValidatesAndStoresFailureMessage(t *testing.T) {
	repository := &deploymentRepositoryFake{
		deployment: Deployment{ID: 1, Status: StatusQueued},
	}
	usecase := NewTransitionDeploymentStatus(repository)

	if _, err := usecase.Execute(context.Background(), 1, StatusFailed, ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("failure without diagnostic error = %v, want ErrValidation", err)
	}
	if _, err := usecase.Execute(context.Background(), 1, StatusBuilding, "not allowed"); !errors.Is(err, ErrValidation) {
		t.Fatalf("diagnostic on non-failure error = %v, want ErrValidation", err)
	}

	if _, err := usecase.Execute(context.Background(), 1, StatusBuilding, ""); err != nil {
		t.Fatalf("transition to building: %v", err)
	}
	failed, err := usecase.Execute(context.Background(), 1, StatusFailed, " build failed ")
	if err != nil {
		t.Fatalf("transition to failed: %v", err)
	}
	if failed.Status != StatusFailed ||
		failed.ErrorMessage == nil ||
		*failed.ErrorMessage != "build failed" {
		t.Fatalf("failure transition = %+v", failed)
	}
}

func TestTransitionDeploymentStatusRejectsConcurrentChange(t *testing.T) {
	repository := &deploymentRepositoryFake{
		deployment:    Deployment{ID: 1, Status: StatusQueued},
		transitionErr: ErrConcurrentTransition,
	}
	_, err := NewTransitionDeploymentStatus(repository).Execute(
		context.Background(),
		1,
		StatusBuilding,
		"",
	)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Execute() error = %v, want ErrInvalidTransition", err)
	}
}
