package applications

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestCreateApplicationAcceptsGitHubRepositoryID(t *testing.T) {
	repositoryID := int64(12345)
	repository := &applicationRepositoryFake{}
	handler := NewHandler(
		NewCreateApplication(repository),
		NewGetApplicationByID(repository),
		NewListApplications(repository),
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
	var application Application
	if err := json.NewDecoder(response.Body).Decode(&application); err != nil {
		t.Fatalf("decode create-application response: %v", err)
	}
	if application.GitHubRepositoryID == nil || *application.GitHubRepositoryID != repositoryID {
		t.Fatalf("response GitHub repository ID = %v, want %d", application.GitHubRepositoryID, repositoryID)
	}
}

type applicationRepositoryFake struct {
	created Application
}

func (repository *applicationRepositoryFake) CreateApplication(
	_ context.Context,
	application Application,
) (Application, error) {
	application.ID = 1
	repository.created = application
	return application, nil
}

func (repository *applicationRepositoryFake) GetApplicationByID(
	context.Context,
	int64,
) (Application, error) {
	return repository.created, nil
}

func (repository *applicationRepositoryFake) GetApplicationByGitHubRepositoryID(
	context.Context,
	int64,
) (Application, error) {
	return repository.created, nil
}

func (repository *applicationRepositoryFake) ListApplications(
	context.Context,
	int,
	int,
) ([]Application, error) {
	if repository.created.ID == 0 {
		return []Application{}, nil
	}
	return []Application{repository.created}, nil
}
