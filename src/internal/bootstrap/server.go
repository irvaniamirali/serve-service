package bootstrap

import (
	"errors"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/deployments"
	githubclient "github.com/irvaniamirali/serve-service/internal/github"
	"github.com/irvaniamirali/serve-service/internal/infrastructure/httpx"
	"gorm.io/gorm"
)

const (
	healthPath   = "/health"
	healthStatus = "healthy"
)

type healthResponse struct {
	Status string `json:"status"`
}

// NewServer wires the API's repositories, use cases, controllers, and routes.
func NewServer(db *gorm.DB) *fiber.App {
	applicationRepository := applications.NewApplicationRepository(db)
	deploymentRepository := deployments.NewDeploymentRepository(db)
	webhookProcessor := newGitHubWebhookProcessor(
		applicationRepository,
		deploymentRepository,
		webhookSecretFromEnvironment(),
		db,
	)
	applicationHTTPHandler := applications.NewHandler(
		applications.NewCreateApplication(applicationRepository),
		applications.NewGetApplicationByID(applicationRepository),
		applications.NewListApplications(applicationRepository),
	)
	deploymentHTTPHandler := deployments.NewHandler(
		deployments.NewCreateDeployment(
			deploymentRepository,
			applicationRepository,
			githubclient.NewClient(os.Getenv("GITHUB_TOKEN")),
		),
		deployments.NewGetDeploymentByID(deploymentRepository),
		deployments.NewListDeploymentsByApplication(deploymentRepository, applicationRepository),
	)

	return newServer(applicationHTTPHandler, deploymentHTTPHandler, webhookProcessor)
}

type routeRegistrar interface {
	RegisterRoutes(fiber.Router)
}

func newServer(
	applicationHTTPHandler routeRegistrar,
	deploymentHTTPHandler routeRegistrar,
	webhookProcessor *GitHubWebhookProcessor,
) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 1 << 20,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			var fiberError *fiber.Error
			if errors.As(err, &fiberError) {
				if fiberError.Code == fiber.StatusNotFound {
					return httpx.WriteError(ctx, fiber.StatusNotFound, "not_found", "Resource not found")
				}
				if fiberError.Code == fiber.StatusRequestEntityTooLarge {
					return httpx.WriteError(ctx, fiber.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds the allowed size")
				}
			}
			slog.Error("unhandled HTTP error", "error", err)
			return httpx.WriteError(ctx, fiber.StatusInternalServerError, "internal_error", "An unexpected error occurred")
		},
	})

	app.Get(healthPath, healthHandler)
	webhookProcessor.registerRoutes(app)
	api := app.Group("/api/v1")
	applicationHTTPHandler.RegisterRoutes(api)
	deploymentHTTPHandler.RegisterRoutes(api)

	return app
}

func healthHandler(c *fiber.Ctx) error {
	response := healthResponse{
		Status: healthStatus,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
