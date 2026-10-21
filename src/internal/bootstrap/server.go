package bootstrap

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	applicationcontroller "github.com/irvaniamirali/serve-service/internal/applications/adapters/http/controllers"
	applicationpostgres "github.com/irvaniamirali/serve-service/internal/applications/adapters/postgres"
	applicationusecase "github.com/irvaniamirali/serve-service/internal/applications/usecase"
	deploymentcontroller "github.com/irvaniamirali/serve-service/internal/deployments/adapters/http/controllers"
	deploymentpostgres "github.com/irvaniamirali/serve-service/internal/deployments/adapters/postgres"
	deploymentusecase "github.com/irvaniamirali/serve-service/internal/deployments/usecase"
	"github.com/irvaniamirali/serve-service/internal/platform/httpx"
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
	applicationHTTPHandler := applicationcontroller.NewHandler(
		applicationusecase.NewCreateApplication(applicationRepository),
		applicationusecase.NewGetApplicationByID(applicationRepository),
	)
	deploymentHTTPHandler := deploymentcontroller.NewHandler(
		deploymentusecase.NewCreateDeployment(deploymentRepository, applicationRepository),
		deploymentusecase.NewGetDeploymentByID(deploymentRepository),
		deploymentusecase.NewListDeploymentsByApplication(deploymentRepository, applicationRepository),
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
