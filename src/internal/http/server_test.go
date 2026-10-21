package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/applications/usecase"
	"github.com/irvaniamirali/serve-service/internal/deployments"
	deploymentusecase "github.com/irvaniamirali/serve-service/internal/deployments/usecase"
	applicationhandler "github.com/irvaniamirali/serve-service/internal/http/handlers/applications"
	deploymenthandler "github.com/irvaniamirali/serve-service/internal/http/handlers/deployments"
)

func TestHealthEndpoint(t *testing.T) {
	app := newTestServer(&httpApplicationRepository{}, &httpDeploymentRepository{})
	request, err := http.NewRequest(http.MethodGet, healthPath, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Errorf("status code = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != healthStatus {
		t.Errorf("status = %q, want %q", body.Status, healthStatus)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, fiber.MIMEApplicationJSON) {
		t.Errorf("Content-Type = %q, want JSON", contentType)
	}
}

func TestApplicationAndDeploymentEndpoints(t *testing.T) {
	app := newTestServer(&httpApplicationRepository{}, &httpDeploymentRepository{})

	response := sendRequest(t, app, http.MethodPost, "/api/v1/applications", `{
		"name": "demo",
		"repository_url": "https://github.com/acme/demo.git",
		"branch": "main"
	}`)
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create application status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	var application applications.Application
	decodeResponse(t, response, &application)
	if application.ID != 1 || application.Name != "demo" || application.Branch != "main" {
		t.Fatalf("unexpected application response: %+v", application)
	}

	response = sendRequest(t, app, http.MethodPost, "/api/v1/applications", `{
		"name": "demo",
		"repository_url": "https://github.com/acme/another.git",
		"branch": "main"
	}`)
	if response.StatusCode != fiber.StatusConflict {
		response.Body.Close()
		t.Fatalf("duplicate application status = %d, want %d", response.StatusCode, fiber.StatusConflict)
	}
	response.Body.Close()

	response = sendRequest(t, app, http.MethodGet, "/api/v1/applications/1", "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("get application status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	decodeResponse(t, response, &application)

	response = sendRequest(t, app, http.MethodPost, "/api/v1/applications/1/deployments", `{"commit_sha":"abcdef0123456789abcdef0123456789abcdef01"}`)
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create deployment status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	var deployment deployments.Deployment
	decodeResponse(t, response, &deployment)
	if deployment.Status != deployments.StatusQueued || deployment.ApplicationID != application.ID {
		t.Fatalf("unexpected deployment response: %+v", deployment)
	}

	response = sendRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/deployments/%d", deployment.ID), "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("get deployment status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	decodeResponse(t, response, &deployment)

	response = sendRequest(t, app, http.MethodGet, "/api/v1/applications/1/deployments", "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("list deployments status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	var history struct {
		Deployments []deployments.Deployment `json:"deployments"`
		Limit       int                      `json:"limit"`
		Offset      int                      `json:"offset"`
	}
	decodeResponse(t, response, &history)
	if len(history.Deployments) != 1 || history.Limit != deployments.DefaultPageSize {
		t.Fatalf("unexpected deployment history: %+v", history)
	}
}

func TestApplicationAndDeploymentEndpointErrors(t *testing.T) {
	app := newTestServer(&httpApplicationRepository{}, &httpDeploymentRepository{})
	createResponse := sendRequest(t, app, http.MethodPost, "/api/v1/applications", `{
		"name": "existing",
		"repository_url": "https://github.com/acme/existing.git",
		"branch": "main"
	}`)
	if createResponse.StatusCode != fiber.StatusCreated {
		createResponse.Body.Close()
		t.Fatalf("create test application status = %d, want %d", createResponse.StatusCode, fiber.StatusCreated)
	}
	createResponse.Body.Close()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "invalid repository URL", method: http.MethodPost, path: "/api/v1/applications", body: `{"name":"demo","repository_url":"ftp://example.com/repo","branch":"main"}`, status: fiber.StatusBadRequest},
		{name: "malformed JSON", method: http.MethodPost, path: "/api/v1/applications", body: `{"name":`, status: fiber.StatusBadRequest},
		{name: "unknown application", method: http.MethodGet, path: "/api/v1/applications/42", status: fiber.StatusNotFound},
		{name: "unknown deployment", method: http.MethodGet, path: "/api/v1/deployments/42", status: fiber.StatusNotFound},
		{name: "unknown application history", method: http.MethodGet, path: "/api/v1/applications/42/deployments", status: fiber.StatusNotFound},
		{name: "invalid application ID", method: http.MethodGet, path: "/api/v1/applications/not-an-id", status: fiber.StatusBadRequest},
		{name: "unknown application deployment request", method: http.MethodPost, path: "/api/v1/applications/42/deployments", status: fiber.StatusNotFound},
		{name: "invalid page size", method: http.MethodGet, path: "/api/v1/applications/42/deployments?limit=101", status: fiber.StatusBadRequest},
		{name: "invalid commit SHA", method: http.MethodPost, path: "/api/v1/applications/1/deployments", body: `{"commit_sha":"not-a-commit"}`, status: fiber.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := sendRequest(t, app, test.method, test.path, test.body)
			defer response.Body.Close()
			if response.StatusCode != test.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.status)
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if _, exists := body["error"]; !exists {
				t.Fatalf("error response has no error field: %v", body)
			}
		})
	}
}

func newTestServer(applicationRepository *httpApplicationRepository, deploymentRepository *httpDeploymentRepository) *fiber.App {
	createApplication := usecase.NewCreateApplication(applicationRepository)
	getApplicationByID := usecase.NewGetApplicationByID(applicationRepository)
	createDeployment := deploymentusecase.NewCreateDeployment(
		deploymentRepository,
		applicationRepository,
	)
	getDeploymentByID := deploymentusecase.NewGetDeploymentByID(deploymentRepository)
	listDeploymentsByApplication := deploymentusecase.NewListDeploymentsByApplication(
		deploymentRepository,
		applicationRepository,
	)
	return NewServer(
		applicationhandler.NewHandler(createApplication, getApplicationByID),
		deploymenthandler.NewHandler(
			createDeployment,
			getDeploymentByID,
			listDeploymentsByApplication,
		),
	)
}

func sendRequest(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return response
}

func decodeResponse(t *testing.T, response *http.Response, destination any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

type httpApplicationRepository struct {
	applicationsByID map[int64]applications.Application
	nextID           int64
}

func (repository *httpApplicationRepository) CreateApplication(_ context.Context, application applications.Application) (applications.Application, error) {
	if repository.applicationsByID == nil {
		repository.applicationsByID = make(map[int64]applications.Application)
	}
	for _, existing := range repository.applicationsByID {
		if existing.Name == application.Name {
			return applications.Application{}, applications.ErrConflict
		}
	}
	repository.nextID++
	application.ID = repository.nextID
	application.CreatedAt = time.Now()
	application.UpdatedAt = application.CreatedAt
	repository.applicationsByID[application.ID] = application
	return application, nil
}

func (repository *httpApplicationRepository) GetApplicationByID(_ context.Context, id int64) (applications.Application, error) {
	application, exists := repository.applicationsByID[id]
	if !exists {
		return applications.Application{}, applications.ErrNotFound
	}
	return application, nil
}

type httpDeploymentRepository struct {
	deploymentsByID map[int64]deployments.Deployment
	nextID          int64
}

func (repository *httpDeploymentRepository) Create(_ context.Context, applicationID int64, commitSHA *string) (deployments.Deployment, error) {
	if repository.deploymentsByID == nil {
		repository.deploymentsByID = make(map[int64]deployments.Deployment)
	}
	repository.nextID++
	now := time.Now()
	deployment := deployments.Deployment{
		ID:            repository.nextID,
		ApplicationID: applicationID,
		Status:        deployments.StatusQueued,
		CommitSHA:     commitSHA,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	repository.deploymentsByID[deployment.ID] = deployment
	return deployment, nil
}

func (repository *httpDeploymentRepository) GetByID(_ context.Context, id int64) (deployments.Deployment, error) {
	deployment, exists := repository.deploymentsByID[id]
	if !exists {
		return deployments.Deployment{}, deployments.ErrNotFound
	}
	return deployment, nil
}

func (repository *httpDeploymentRepository) ListByApplication(_ context.Context, applicationID int64, limit, offset int) ([]deployments.Deployment, error) {
	result := make([]deployments.Deployment, 0)
	for _, deployment := range repository.deploymentsByID {
		if deployment.ApplicationID == applicationID {
			result = append(result, deployment)
		}
	}
	return result, nil
}

func (repository *httpDeploymentRepository) TransitionStatus(_ context.Context, id int64, expectedStatus, nextStatus string, failureMessage *string) (deployments.Deployment, error) {
	deployment, exists := repository.deploymentsByID[id]
	if !exists {
		return deployments.Deployment{}, deployments.ErrNotFound
	}
	if deployment.Status != expectedStatus {
		return deployments.Deployment{}, deployments.ErrInvalidTransition
	}
	deployment.Status = nextStatus
	deployment.ErrorMessage = failureMessage
	repository.deploymentsByID[id] = deployment
	return deployment, nil
}
