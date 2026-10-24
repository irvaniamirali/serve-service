package bootstrap

import (
	"errors"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	applicationcontroller "github.com/irvaniamirali/serve-service/internal/domains/applications/adapters/http/controllers"
	applicationpostgres "github.com/irvaniamirali/serve-service/internal/domains/applications/adapters/postgres"
	"github.com/irvaniamirali/serve-service/internal/domains/applications/usecase"
	deploymentcontroller "github.com/irvaniamirali/serve-service/internal/domains/deployments/adapters/http/controllers"
	deploymentpostgres "github.com/irvaniamirali/serve-service/internal/domains/deployments/adapters/postgres"
	usecase2 "github.com/irvaniamirali/serve-service/internal/domains/deployments/usecase"
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
	applicationRepository := applicationpostgres.NewRepository(db)
	deploymentRepository := deploymentpostgres.NewRepository(db)
	setGitHubWebhookProcessor(
		applicationRepository,
		deploymentRepository,
		webhookSecretFromEnvironment(),
		db,
	)
	applicationHTTPHandler := applicationcontroller.NewHandler(
		usecase.NewCreateApplication(applicationRepository),
		usecase.NewGetApplicationByID(applicationRepository),
		usecase.NewListApplications(applicationRepository),
	)
	deploymentHTTPHandler := deploymentcontroller.NewHandler(
		usecase2.NewCreateDeployment(
			deploymentRepository,
			applicationRepository,
			githubclient.NewClient(os.Getenv("GITHUB_TOKEN")),
		),
		usecase2.NewGetDeploymentByID(deploymentRepository),
		usecase2.NewListDeploymentsByApplication(deploymentRepository, applicationRepository),
	)

	return newServer(applicationHTTPHandler, deploymentHTTPHandler)
}

type routeRegistrar interface {
	RegisterRoutes(fiber.Router)
}

func newServer(applicationHTTPHandler, deploymentHTTPHandler routeRegistrar) *fiber.App {
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
	registerGitHubWebhookRoute(app)
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
