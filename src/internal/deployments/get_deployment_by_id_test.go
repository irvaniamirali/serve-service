package deployments

import (
	"context"
	"errors"
	"testing"
)

func TestGetDeploymentByIDReturnsRepositoryResult(t *testing.T) {
	want := Deployment{ID: 4, ApplicationID: 9, Status: StatusQueued}
	repository := &deploymentRepositoryFake{deployment: want}

	got, err := NewGetDeploymentByID(repository).Execute(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != want {
		t.Fatalf("Execute() = %+v, want %+v", got, want)
	}
	if repository.getCalls != 1 {
		t.Fatalf("GetByID calls = %d, want 1", repository.getCalls)
	}
}

func TestGetDeploymentByIDRejectsInvalidIDWithoutRepositoryCall(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	_, err := NewGetDeploymentByID(repository).Execute(context.Background(), 0)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Execute() error = %v, want ErrNotFound", err)
	}
	if repository.getCalls != 0 {
		t.Fatalf("GetByID calls = %d, want 0", repository.getCalls)
	}
}
