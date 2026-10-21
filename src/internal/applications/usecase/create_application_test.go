package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

func TestCreateApplicationValidatesAndPersists(t *testing.T) {
	repository := &createApplicationRepository{}
	create := NewCreateApplication(repository)
	input := CreateApplicationInput{
		Name:          " demo ",
		RepositoryURL: "https://github.com/acme/demo.git",
		Branch:        " main ",
	}

	application, err := create.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if application.Name != input.Name ||
		application.RepositoryURL != input.RepositoryURL ||
		application.Branch != input.Branch {
		t.Fatalf("persisted application = %+v, want input values", application)
	}
	if repository.createCalls != 1 {
		t.Fatalf("CreateApplication calls = %d, want 1", repository.createCalls)
	}
}

func TestCreateApplicationRejectsInvalidInputWithoutPersistence(t *testing.T) {
	repository := &createApplicationRepository{}
	create := NewCreateApplication(repository)

	_, err := create.Execute(context.Background(), CreateApplicationInput{
		Name:          "demo",
		RepositoryURL: "ftp://example.com/demo.git",
		Branch:        "main",
	})
	if !errors.Is(err, applications.ErrValidation) {
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
	application applications.Application,
) (applications.Application, error) {
	repository.createCalls++
	return application, nil
}

func (*createApplicationRepository) GetApplicationByID(
	context.Context,
	int64,
) (applications.Application, error) {
	return applications.Application{}, applications.ErrNotFound
}
