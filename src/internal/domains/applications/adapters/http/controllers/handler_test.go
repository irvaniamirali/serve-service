package controllers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/irvaniamirali/serve-service/internal/domains/applications"
	usecase2 "github.com/irvaniamirali/serve-service/internal/domains/applications/usecase"
)

func TestCreateApplicationAcceptsGitHubRepositoryID(t *testing.T) {
	repositoryID := int64(12345)
	repository := &applicationRepositoryFake{}
	handler := NewHandler(
		usecase2.NewCreateApplication(repository),
		usecase2.NewGetApplicationByID(repository),
		usecase2.NewListApplications(repository),
	)
	app := fiber.New()
	handler.RegisterRoutes(app)

	request := httptest.NewRequest(
		"POST",
		"/applications",
		strings.NewReader(`{
			"name": "example",
			"repository_url": "https://github.com/example/app.git",
			"branch": "main",
			"github_repository_id": 12345
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send create-application request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	if repository.created.GitHubRepositoryID == nil || *repository.created.GitHubRepositoryID != repositoryID {
		t.Fatalf("persisted GitHub repository ID = %v, want %d", repository.created.GitHubRepositoryID, repositoryID)
	}
	var application applications.Application
	if err := json.NewDecoder(response.Body).Decode(&application); err != nil {
		t.Fatalf("decode create-application response: %v", err)
	}
	if application.GitHubRepositoryID == nil || *application.GitHubRepositoryID != repositoryID {
		t.Fatalf("response GitHub repository ID = %v, want %d", application.GitHubRepositoryID, repositoryID)
	}
}

type applicationRepositoryFake struct {
	created applications.Application
}

func (repository *applicationRepositoryFake) CreateApplication(
	_ context.Context,
	application applications.Application,
) (applications.Application, error) {
	application.ID = 1
	repository.created = application
	return application, nil
}

func (repository *applicationRepositoryFake) GetApplicationByID(
	context.Context,
	int64,
) (applications.Application, error) {
	return repository.created, nil
}

func (repository *applicationRepositoryFake) GetApplicationByGitHubRepositoryID(
	context.Context,
	int64,
) (applications.Application, error) {
	return repository.created, nil
}

func (repository *applicationRepositoryFake) ListApplications(
	context.Context,
	int,
	int,
) ([]applications.Application, error) {
	if repository.created.ID == 0 {
		return []applications.Application{}, nil
	}
	return []applications.Application{repository.created}, nil
}
