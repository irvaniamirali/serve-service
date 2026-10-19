package http

import "github.com/gofiber/fiber/v2"

const (
	healthPath   = "/health"
	healthStatus = "healthy"
)

type healthResponse struct {
	Status string `json:"status"`
}

func NewServer() *fiber.App {
	app := fiber.New()

	app.Get(healthPath, healthHandler)

	return app
}

func healthHandler(c *fiber.Ctx) error {
	response := healthResponse{
		Status: healthStatus,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
