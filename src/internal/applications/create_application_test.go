package applications

import (
	"context"
	"errors"
	"testing"
)

func TestCreateApplicationValidatesAndPersists(t *testing.T) {
	repository := &createApplicationRepository{}
	create := NewCreateApplication(repository)
	params := CreateApplicationParams{
		Name:          " demo ",
		RepositoryURL: "https://github.com/acme/demo.git",
		Branch:        " main ",
	}

	application, err := create.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if application.Name != params.Name ||
		application.RepositoryURL != params.RepositoryURL ||
		application.Branch != params.Branch {
		t.Fatalf("persisted application = %+v, want input values", application)
	}
	if repository.createCalls != 1 {
		t.Fatalf("CreateApplication calls = %d, want 1", repository.createCalls)
	}
}

func TestCreateApplicationRejectsInvalidInputWithoutPersistence(t *testing.T) {
	repository := &createApplicationRepository{}
	create := NewCreateApplication(repository)

	_, err := create.Execute(context.Background(), CreateApplicationParams{
		Name:          "demo",
		RepositoryURL: "ftp://example.com/demo.git",
		Branch:        "main",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Execute() error = %v, want ErrValidation", err)
	}
	if repository.createCalls != 0 {
		t.Fatalf("CreateApplication calls = %d, want 0", repository.createCalls)
	}
}

type createApplicationRepository struct {
	createCalls int
}

func (repository *createApplicationRepository) CreateApplication(
	_ context.Context,
	application Application,
) (Application, error) {
	repository.createCalls++
	return application, nil
}

func (*createApplicationRepository) GetApplicationByID(
	context.Context,
	int64,
) (Application, error) {
	return Application{}, ErrNotFound
}

func (*createApplicationRepository) GetApplicationByGitHubRepositoryID(
	context.Context,
	int64,
) (Application, error) {
	return Application{}, ErrNotFound
}

func (*createApplicationRepository) ListApplications(context.Context, int, int) ([]Application, error) {
	return nil, nil
}
