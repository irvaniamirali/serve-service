package http

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/irvaniamirali/serve-service/internal/httpx"
)

const (
	healthPath   = "/health"
	healthStatus = "healthy"
)

type healthResponse struct {
	Status string `json:"status"`
}

type routeRegistrar interface {
	RegisterRoutes(fiber.Router)
}

func NewServer(applicationHandler, deploymentHandler routeRegistrar) *fiber.App {
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
	applicationHandler.RegisterRoutes(api)
	deploymentHandler.RegisterRoutes(api)

	return app
}

func healthHandler(c *fiber.Ctx) error {
	response := healthResponse{
		Status: healthStatus,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
