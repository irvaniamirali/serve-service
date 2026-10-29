package applications

import (
	"context"
	"errors"
	"testing"
)

func TestGetApplicationByIDReturnsRepositoryResult(t *testing.T) {
	want := Application{ID: 8, Name: "demo"}
	repository := &getApplicationRepositoryFake{application: want}

	got, err := NewGetApplicationByID(repository).Execute(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != want {
		t.Fatalf("Execute() = %+v, want %+v", got, want)
	}
	if repository.getCalls != 1 {
		t.Fatalf("GetApplicationByID calls = %d, want 1", repository.getCalls)
	}
}

func TestGetApplicationByIDRejectsInvalidIDWithoutRepositoryCall(t *testing.T) {
	repository := &getApplicationRepositoryFake{}
	_, err := NewGetApplicationByID(repository).Execute(context.Background(), 0)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Execute() error = %v, want ErrNotFound", err)
	}
	if repository.getCalls != 0 {
		t.Fatalf("GetApplicationByID calls = %d, want 0", repository.getCalls)
	}
}

func TestGetApplicationByIDPreservesRepositoryError(t *testing.T) {
	repository := &getApplicationRepositoryFake{err: errors.New("database unavailable")}
	_, err := NewGetApplicationByID(repository).Execute(context.Background(), 1)
	if !errors.Is(err, repository.err) {
		t.Fatalf("Execute() error = %v, want wrapped repository error", err)
	}
}

type getApplicationRepositoryFake struct {
	application Application
	err         error
	getCalls    int
}

func (repository *getApplicationRepositoryFake) CreateApplication(
	context.Context,
	Application,
) (Application, error) {
	return Application{}, errors.New("not used")
}

func (repository *getApplicationRepositoryFake) GetApplicationByID(
	_ context.Context,
	id int64,
) (Application, error) {
	repository.getCalls++
	if repository.err != nil {
		return Application{}, repository.err
	}
	if repository.application.ID != id {
		return Application{}, ErrNotFound
	}
	return repository.application, nil
}

func (repository *getApplicationRepositoryFake) GetApplicationByGitHubRepositoryID(
	context.Context,
	int64,
) (Application, error) {
	return Application{}, errors.New("not used")
}

func (repository *getApplicationRepositoryFake) ListApplications(
	context.Context,
	int,
	int,
) ([]Application, error) {
	return nil, errors.New("not used")
}
