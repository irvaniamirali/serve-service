package controllers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	applicationdomain "github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/applications/usecase"
	"github.com/irvaniamirali/serve-service/internal/platform/httpx"
)

// Handler translates application HTTP requests into use case calls.
type Handler struct {
	createApplication  *usecase.CreateApplication
	getApplicationByID *usecase.GetApplicationByID
}

// NewHandler creates an HTTP handler with the required use cases.
func NewHandler(
	createApplication *usecase.CreateApplication,
	getApplicationByID *usecase.GetApplicationByID,
) *Handler {
	return &Handler{
		createApplication:  createApplication,
		getApplicationByID: getApplicationByID,
	}
}

// RegisterRoutes registers the application endpoints.
func (handler *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/applications", handler.create)
	router.Get("/applications/:id", handler.get)
}

type createRequest struct {
	Name          string `json:"name"`
	RepositoryURL string `json:"repository_url"`
	Branch        string `json:"branch"`
}

func (handler *Handler) create(ctx *fiber.Ctx) error {
	var request createRequest

	if err := httpx.DecodeJSON(ctx, &request); err != nil {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			"Request body must be valid JSON with supported fields",
		)
	}

	application, err := handler.createApplication.Execute(
		ctx.UserContext(),
		usecase.CreateApplicationInput{
			Name:          request.Name,
			RepositoryURL: request.RepositoryURL,
			Branch:        request.Branch,
		},
	)
	if errors.Is(err, applicationdomain.ErrValidation) {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"validation_error",
			err.Error(),
		)
	}
	if errors.Is(err, applicationdomain.ErrConflict) {
		return httpx.WriteError(
			ctx,
			fiber.StatusConflict,
			"application_name_conflict",
			"An application with this name already exists",
		)
	}
	if err != nil {
		return httpx.WriteInternalError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(application)
}

func (handler *Handler) get(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			"Application ID must be a positive integer",
		)
	}

	application, err := handler.getApplicationByID.Execute(
		ctx.UserContext(),
		id,
	)
	if errors.Is(err, applicationdomain.ErrNotFound) {
		return httpx.WriteError(
			ctx,
			fiber.StatusNotFound,
			"application_not_found",
			"Application not found",
		)
	}
	if err != nil {
		return httpx.WriteInternalError(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(application)
}
