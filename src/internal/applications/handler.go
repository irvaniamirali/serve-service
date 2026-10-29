package applications

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/irvaniamirali/serve-service/internal/infrastructure/httpx"
)

// Handler translates application HTTP requests into use case calls.
type Handler struct {
	createApplication  *CreateApplication
	getApplicationByID *GetApplicationByID
	listApplications   *ListApplications
}

// NewHandler creates an HTTP handler with the required use cases.
func NewHandler(
	createApplication *CreateApplication,
	getApplicationByID *GetApplicationByID,
	listApplications *ListApplications,
) *Handler {
	return &Handler{
		createApplication:  createApplication,
		getApplicationByID: getApplicationByID,
		listApplications:   listApplications,
	}
}

// RegisterRoutes registers the application endpoints.
func (handler *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/applications", handler.create)
	router.Get("/applications", handler.list)
	router.Get("/applications/:id", handler.get)
}

func (handler *Handler) create(ctx *fiber.Ctx) error {
	var request CreateApplicationRequest

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
		CreateApplicationParams{
			Name:               request.Name,
			RepositoryURL:      request.RepositoryURL,
			Branch:             request.Branch,
			GitHubRepositoryID: request.GitHubRepositoryID,
		},
	)
	if errors.Is(err, ErrValidation) {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"validation_error",
			err.Error(),
		)
	}
	if errors.Is(err, ErrConflict) {
		return httpx.WriteError(
			ctx,
			fiber.StatusConflict,
			"application_conflict",
			"An application with this name or GitHub repository is already registered",
		)
	}
	if err != nil {
		return httpx.WriteInternalError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(application)
}

func (handler *Handler) get(ctx *fiber.Ctx) error {
	applicationID, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil || applicationID < 1 {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			"Application ID must be a positive integer",
		)
	}

	application, err := handler.getApplicationByID.Execute(
		ctx.UserContext(),
		applicationID,
	)
	if errors.Is(err, ErrNotFound) {
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

func (handler *Handler) list(ctx *fiber.Ctx) error {
	limit, err := parseQueryInteger(
		ctx.Query("limit"),
		DefaultApplicationPageSize,
	)
	if err != nil || limit < 1 || limit > MaxApplicationPageSize {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			"limit must be an integer between 1 and 100",
		)
	}

	offset, err := parseQueryInteger(ctx.Query("offset"), 0)
	if err != nil || offset < 0 {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			"offset must be a non-negative integer",
		)
	}

	applicationList, err := handler.listApplications.Execute(
		ctx.UserContext(),
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			return httpx.WriteError(
				ctx,
				fiber.StatusBadRequest,
				"invalid_request",
				err.Error(),
			)
		}
		return httpx.WriteInternalError(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(ListApplicationsResponse{
		Applications: applicationList,
		Limit:        limit,
		Offset:       offset,
	})
}
