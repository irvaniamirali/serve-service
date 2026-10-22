package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	applicationdomain "github.com/irvaniamirali/serve-service/internal/applications"
	applicationhandler "github.com/irvaniamirali/serve-service/internal/applications/adapters/http/controllers"
	"github.com/irvaniamirali/serve-service/internal/applications/usecase"
	deploymentdomain "github.com/irvaniamirali/serve-service/internal/deployments"
	deploymenthandler "github.com/irvaniamirali/serve-service/internal/deployments/adapters/http/controllers"
	deploymentusecase "github.com/irvaniamirali/serve-service/internal/deployments/usecase"
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
	defer closeResponse(t, response)

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
	var application applicationdomain.Application
	decodeResponse(t, response, &application)
	if application.ID != 1 || application.Name != "demo" || application.Branch != "main" {
		t.Fatalf("unexpected application response: %+v", application)
	}
	response = sendRequest(t, app, http.MethodGet, "/api/v1/applications?limit=10&offset=0", "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("list applications status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	var applicationList struct {
		Applications []applicationdomain.Application `json:"applications"`
		Limit        int                             `json:"limit"`
		Offset       int                             `json:"offset"`
	}
	decodeResponse(t, response, &applicationList)
	if len(applicationList.Applications) != 1 || applicationList.Applications[0].ID != application.ID ||
		applicationList.Limit != 10 || applicationList.Offset != 0 {
		t.Fatalf("unexpected application list: %+v", applicationList)
	}

	response = sendRequest(t, app, http.MethodPost, "/api/v1/applications", `{
		"name": "demo",
		"repository_url": "https://github.com/acme/another.git",
		"branch": "main"
	}`)
	if response.StatusCode != fiber.StatusConflict {
		closeResponse(t, response)
		t.Fatalf("duplicate application status = %d, want %d", response.StatusCode, fiber.StatusConflict)
	}
	closeResponse(t, response)

	response = sendRequest(t, app, http.MethodGet, "/api/v1/applications/1", "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("get application status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	decodeResponse(t, response, &application)

	response = sendRequest(t, app, http.MethodPost, "/api/v1/applications/1/deployments", `{}`)
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create default-branch deployment status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	var defaultDeployment deploymentdomain.Deployment
	decodeResponse(t, response, &defaultDeployment)
	if defaultDeployment.CommitSHA == nil || *defaultDeployment.CommitSHA != "abcdef0123456789abcdef0123456789abcdef01" {
		t.Fatalf("default-branch deployment SHA = %v, want resolved immutable SHA", defaultDeployment.CommitSHA)
	}

	response = sendRequest(t, app, http.MethodPost, "/api/v1/applications/1/deployments", `{"commit_sha":"abcdef0123456789abcdef0123456789abcdef01"}`)
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create deployment status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	var deployment deploymentdomain.Deployment
	decodeResponse(t, response, &deployment)
	response = sendRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/deployments/%d/logs", deployment.ID), "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("get deployment logs status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	var buildLog struct {
		DeploymentID int64  `json:"deployment_id"`
		Log          string `json:"log"`
	}
	decodeResponse(t, response, &buildLog)
	if buildLog.DeploymentID != deployment.ID || buildLog.Log != "" {
		t.Fatalf("unexpected deployment logs response: %+v", buildLog)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("log cache policy = %q, want no-store", response.Header.Get("Cache-Control"))
	}
	if deployment.Status != deploymentdomain.StatusQueued || deployment.ApplicationID != application.ID {
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
		Deployments []deploymentdomain.Deployment `json:"deployments"`
		Limit       int                           `json:"limit"`
		Offset      int                           `json:"offset"`
	}
	decodeResponse(t, response, &history)
	if len(history.Deployments) != 2 || history.Limit != deploymentdomain.DefaultPageSize {
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
		closeResponse(t, createResponse)
		t.Fatalf("create test application status = %d, want %d", createResponse.StatusCode, fiber.StatusCreated)
	}
	closeResponse(t, createResponse)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "invalid repository URL", method: http.MethodPost, path: "/api/v1/applications", body: `{"name":"demo","repository_url":"ftp://example.com/repo","branch":"main"}`, status: fiber.StatusBadRequest},
		{name: "invalid application page size", method: http.MethodGet, path: "/api/v1/applications?limit=101", status: fiber.StatusBadRequest},
		{name: "malformed JSON", method: http.MethodPost, path: "/api/v1/applications", body: `{"name":`, status: fiber.StatusBadRequest},
		{name: "unknown application", method: http.MethodGet, path: "/api/v1/applications/42", status: fiber.StatusNotFound},
		{name: "unknown deployment", method: http.MethodGet, path: "/api/v1/deployments/42", status: fiber.StatusNotFound},
		{name: "unknown deployment logs", method: http.MethodGet, path: "/api/v1/deployments/42/logs", status: fiber.StatusNotFound},
		{name: "unknown application history", method: http.MethodGet, path: "/api/v1/applications/42/deployments", status: fiber.StatusNotFound},
		{name: "invalid application ID", method: http.MethodGet, path: "/api/v1/applications/not-an-id", status: fiber.StatusBadRequest},
		{name: "unknown application deployment request", method: http.MethodPost, path: "/api/v1/applications/42/deployments", status: fiber.StatusNotFound},
		{name: "invalid page size", method: http.MethodGet, path: "/api/v1/applications/42/deployments?limit=101", status: fiber.StatusBadRequest},
		{name: "invalid commit SHA", method: http.MethodPost, path: "/api/v1/applications/1/deployments", body: `{"commit_sha":"not-a-commit"}`, status: fiber.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := sendRequest(t, app, test.method, test.path, test.body)
			defer closeResponse(t, response)
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

func TestDeploymentCreationReturnsGatewayErrorWhenDefaultCommitCannotBeResolved(t *testing.T) {
	app := newTestServerWithResolver(
		&httpApplicationRepository{},
		&httpDeploymentRepository{},
		defaultCommitResolverFake{err: errors.New("GitHub API rate limit exceeded")},
	)
	response := sendRequest(t, app, http.MethodPost, "/api/v1/applications", `{
		"name": "demo",
		"repository_url": "https://github.com/acme/demo.git",
		"branch": "release"
	}`)
	if response.StatusCode != fiber.StatusCreated {
		closeResponse(t, response)
		t.Fatalf("create test application status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	closeResponse(t, response)

	response = sendRequest(t, app, http.MethodPost, "/api/v1/applications/1/deployments", `{}`)
	defer closeResponse(t, response)
	if response.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("create default-branch deployment status = %d, want %d", response.StatusCode, fiber.StatusBadGateway)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode resolution error: %v", err)
	}
	if body.Error.Code != "commit_resolution_failed" || !strings.Contains(body.Error.Message, "default branch") {
		t.Fatalf("resolution error response = %+v", body)
	}
}

func TestDeploymentLogsEndpointReturnsPlainJSONTextWithoutAddingLogsToDeployment(t *testing.T) {
	buildLog := "Step 1/1 : FROM scratch\nSuccessfully built"
	deploymentRepository := &httpDeploymentRepository{
		deploymentsByID: map[int64]deploymentdomain.Deployment{
			9: {
				ID:       9,
				Status:   deploymentdomain.StatusBuilt,
				BuildLog: &buildLog,
			},
		},
	}
	app := newTestServer(&httpApplicationRepository{}, deploymentRepository)

	response := sendRequest(t, app, http.MethodGet, "/api/v1/deployments/9/logs", "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("log endpoint status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	var result struct {
		DeploymentID int64  `json:"deployment_id"`
		Log          string `json:"log"`
	}
	decodeResponse(t, response, &result)
	if result.DeploymentID != 9 || result.Log != buildLog {
		t.Fatalf("log response = %+v, want deployment 9 and original text", result)
	}

	response = sendRequest(t, app, http.MethodGet, "/api/v1/deployments/9", "")
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("deployment endpoint status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	var deploymentResponse map[string]json.RawMessage
	decodeResponse(t, response, &deploymentResponse)
	if _, exists := deploymentResponse["build_log"]; exists {
		t.Fatal("deployment JSON unexpectedly includes build logs")
	}
}

func TestGitHubWebhookPushCreatesDeployment(t *testing.T) {
	applicationRepository := &httpApplicationRepository{}
	deploymentRepository := &httpDeploymentRepository{}
	ghRepositoryID := int64(42)
	applicationRepository.applicationsByID = map[int64]applicationdomain.Application{
		1: {
			ID:                 1,
			Name:               "demo",
			RepositoryURL:      "https://github.com/acme/demo.git",
			Branch:             "main",
			GitHubRepositoryID: &ghRepositoryID,
		},
	}
	applicationRepository.nextID = 1
	app := newTestServer(applicationRepository, deploymentRepository)

	payload := `{"ref":"refs/heads/main","after":"abcdef0123456789abcdef0123456789abcdef01","repository":{"id":42,"full_name":"acme/demo"}}`
	signature := "sha256=" + signHMAC("test-secret", payload)
	request, err := http.NewRequest(http.MethodPost, "/api/v1/webhooks/github", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Delivery", "delivery-1")
	request.Header.Set("X-Hub-Signature-256", signature)

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer closeResponse(t, response)
	if response.StatusCode != fiber.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusAccepted)
	}
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["status"] != "processed" {
		t.Fatalf("status response = %v, want processed", result)
	}
	if len(deploymentRepository.deploymentsByID) != 1 {
		t.Fatalf("deployment count = %d, want 1", len(deploymentRepository.deploymentsByID))
	}
	for _, deployment := range deploymentRepository.deploymentsByID {
		if deployment.CommitSHA == nil || *deployment.CommitSHA != "abcdef0123456789abcdef0123456789abcdef01" {
			t.Fatalf("webhook deployment SHA = %v, want push event SHA", deployment.CommitSHA)
		}
	}
}

func TestGitHubWebhookRejectsInvalidSignature(t *testing.T) {
	applicationRepository := &httpApplicationRepository{}
	deploymentRepository := &httpDeploymentRepository{}
	app := newTestServer(applicationRepository, deploymentRepository)
	payload := `{"ref":"refs/heads/main","after":"abcdef0123456789abcdef0123456789abcdef01","repository":{"id":8,"full_name":"acme/demo"}}`
	request, err := http.NewRequest(http.MethodPost, "/api/v1/webhooks/github", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Delivery", "delivery-2")
	request.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer closeResponse(t, response)
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
	}
}

func newTestServer(applicationRepository *httpApplicationRepository, deploymentRepository *httpDeploymentRepository) *fiber.App {
	return newTestServerWithResolver(
		applicationRepository,
		deploymentRepository,
		defaultCommitResolverFake{sha: "abcdef0123456789abcdef0123456789abcdef01"},
	)
}

func newTestServerWithResolver(
	applicationRepository *httpApplicationRepository,
	deploymentRepository *httpDeploymentRepository,
	commitResolver deploymentusecase.DefaultCommitResolver,
) *fiber.App {
	setGitHubWebhookProcessor(applicationRepository, deploymentRepository, "test-secret", nil)
	createApplication := usecase.NewCreateApplication(applicationRepository)
	getApplicationByID := usecase.NewGetApplicationByID(applicationRepository)
	listApplications := usecase.NewListApplications(applicationRepository)
	createDeployment := deploymentusecase.NewCreateDeployment(
		deploymentRepository,
		applicationRepository,
		commitResolver,
	)
	getDeploymentByID := deploymentusecase.NewGetDeploymentByID(deploymentRepository)
	listDeploymentsByApplication := deploymentusecase.NewListDeploymentsByApplication(
		deploymentRepository,
		applicationRepository,
	)
	return newServer(
		applicationhandler.NewHandler(createApplication, getApplicationByID, listApplications),
		deploymenthandler.NewHandler(
			createDeployment,
			getDeploymentByID,
			listDeploymentsByApplication,
		),
	)
}

type defaultCommitResolverFake struct {
	sha string
	err error
}

func (resolver defaultCommitResolverFake) ResolveDefaultBranchHead(
	context.Context,
	applicationdomain.Application,
) (string, error) {
	return resolver.sha, resolver.err
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
	defer closeResponse(t, response)
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func closeResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
}

func signHMAC(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

type httpApplicationRepository struct {
	applicationsByID map[int64]applicationdomain.Application
	nextID           int64
}

func (repository *httpApplicationRepository) CreateApplication(_ context.Context, application applicationdomain.Application) (applicationdomain.Application, error) {
	if repository.applicationsByID == nil {
		repository.applicationsByID = make(map[int64]applicationdomain.Application)
	}
	for _, existing := range repository.applicationsByID {
		if existing.Name == application.Name {
			return applicationdomain.Application{}, applicationdomain.ErrConflict
		}
	}
	repository.nextID++
	application.ID = repository.nextID
	application.CreatedAt = time.Now()
	application.UpdatedAt = application.CreatedAt
	repository.applicationsByID[application.ID] = application
	return application, nil
}

func (repository *httpApplicationRepository) GetApplicationByID(_ context.Context, id int64) (applicationdomain.Application, error) {
	application, exists := repository.applicationsByID[id]
	if !exists {
		return applicationdomain.Application{}, applicationdomain.ErrNotFound
	}
	return application, nil
}

func (repository *httpApplicationRepository) GetApplicationByGitHubRepositoryID(_ context.Context, githubRepositoryID int64) (applicationdomain.Application, error) {
	for _, application := range repository.applicationsByID {
		if application.GitHubRepositoryID != nil && *application.GitHubRepositoryID == githubRepositoryID {
			return application, nil
		}
	}
	return applicationdomain.Application{}, applicationdomain.ErrNotFound
}

func (repository *httpApplicationRepository) ListApplications(_ context.Context, limit, offset int) ([]applicationdomain.Application, error) {
	result := make([]applicationdomain.Application, 0, len(repository.applicationsByID))
	for _, application := range repository.applicationsByID {
		result = append(result, application)
	}
	if offset >= len(result) {
		return []applicationdomain.Application{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

type httpDeploymentRepository struct {
	deploymentsByID map[int64]deploymentdomain.Deployment
	nextID          int64
}

func (repository *httpDeploymentRepository) Create(_ context.Context, applicationID int64, commitSHA *string) (deploymentdomain.Deployment, error) {
	if repository.deploymentsByID == nil {
		repository.deploymentsByID = make(map[int64]deploymentdomain.Deployment)
	}
	repository.nextID++
	now := time.Now()
	deployment := deploymentdomain.Deployment{
		ID:            repository.nextID,
		ApplicationID: applicationID,
		Status:        deploymentdomain.StatusQueued,
		CommitSHA:     commitSHA,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	repository.deploymentsByID[deployment.ID] = deployment
	return deployment, nil
}

func (repository *httpDeploymentRepository) GetByID(_ context.Context, id int64) (deploymentdomain.Deployment, error) {
	deployment, exists := repository.deploymentsByID[id]
	if !exists {
		return deploymentdomain.Deployment{}, deploymentdomain.ErrNotFound
	}
	return deployment, nil
}

func (repository *httpDeploymentRepository) ListByApplication(_ context.Context, applicationID int64, limit, offset int) ([]deploymentdomain.Deployment, error) {
	result := make([]deploymentdomain.Deployment, 0)
	for _, deployment := range repository.deploymentsByID {
		if deployment.ApplicationID == applicationID {
			result = append(result, deployment)
		}
	}
	return result, nil
}

func (repository *httpDeploymentRepository) TransitionStatus(_ context.Context, id int64, expectedStatus, nextStatus string, failureMessage *string) (deploymentdomain.Deployment, error) {
	deployment, exists := repository.deploymentsByID[id]
	if !exists {
		return deploymentdomain.Deployment{}, deploymentdomain.ErrNotFound
	}
	if deployment.Status != expectedStatus {
		return deploymentdomain.Deployment{}, deploymentdomain.ErrInvalidTransition
	}
	deployment.Status = nextStatus
	deployment.ErrorMessage = failureMessage
	repository.deploymentsByID[id] = deployment
	return deployment, nil
}
