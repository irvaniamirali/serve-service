package controllers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	applicationdomain "github.com/irvaniamirali/serve-service/internal/applications"
	deploymentdomain "github.com/irvaniamirali/serve-service/internal/deployments"
	"github.com/irvaniamirali/serve-service/internal/deployments/usecase"
	"github.com/irvaniamirali/serve-service/internal/platform/httpx"
)

// Handler translates deployment HTTP requests into use case calls.
type Handler struct {
	createDeployment             *usecase.CreateDeployment
	getDeploymentByID            *usecase.GetDeploymentByID
	listDeploymentsByApplication *usecase.ListDeploymentsByApplication
}

func NewHandler(
	createDeployment *usecase.CreateDeployment,
	getDeploymentByID *usecase.GetDeploymentByID,
	listDeploymentsByApplication *usecase.ListDeploymentsByApplication,
) *Handler {
	return &Handler{
		createDeployment:             createDeployment,
		getDeploymentByID:            getDeploymentByID,
		listDeploymentsByApplication: listDeploymentsByApplication,
	}
}

func (handler *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/applications/:id/deployments", handler.create)
	router.Get("/applications/:id/deployments", handler.listByApplication)
	router.Get("/deployments/:id", handler.get)
}

type createRequest struct {
	CommitSHA string `json:"commit_sha"`
}

type listResponse struct {
	Deployments []deploymentdomain.Deployment `json:"deployments"`
	Limit       int                           `json:"limit"`
	Offset      int                           `json:"offset"`
}

func (handler *Handler) create(ctx *fiber.Ctx) error {
	applicationID, err := parsePositiveID(ctx.Params("id"), "Application")
	if err != nil {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_request", err.Error())
	}
	var request createRequest
	if len(ctx.Body()) > 0 {
		if err := httpx.DecodeJSON(ctx, &request); err != nil {
			return httpx.WriteError(
				ctx,
				fiber.StatusBadRequest,
				"invalid_request",
				"Request body must be valid JSON with supported fields",
			)
		}
	}
	deployment, err := handler.createDeployment.Execute(
		ctx.UserContext(),
		applicationID,
		request.CommitSHA,
	)
	if errors.Is(err, deploymentdomain.ErrValidation) {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "validation_error", err.Error())
	}
	if errors.Is(err, applicationdomain.ErrNotFound) {
		return httpx.WriteError(ctx, fiber.StatusNotFound, "application_not_found", "Application not found")
	}
	if err != nil {
		return httpx.WriteInternalError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(deployment)
}

func (handler *Handler) get(ctx *fiber.Ctx) error {
	id, err := parsePositiveID(ctx.Params("id"), "Deployment")
	if err != nil {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_request", err.Error())
	}
	deployment, err := handler.getDeploymentByID.Execute(ctx.UserContext(), id)
	if errors.Is(err, deploymentdomain.ErrNotFound) {
		return httpx.WriteError(ctx, fiber.StatusNotFound, "deployment_not_found", "Deployment not found")
	}
	if err != nil {
		return httpx.WriteInternalError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(deployment)
}

func (handler *Handler) listByApplication(ctx *fiber.Ctx) error {
	applicationID, err := parsePositiveID(ctx.Params("id"), "Application")
	if err != nil {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_request", err.Error())
	}
	limit, err := parseQueryInteger(ctx.Query("limit"), deploymentdomain.DefaultPageSize)
	if err != nil || limit < 1 || limit > deploymentdomain.MaxPageSize {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_request", "limit must be an integer between 1 and 100")
	}
	offset, err := parseQueryInteger(ctx.Query("offset"), 0)
	if err != nil || offset < 0 {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_request", "offset must be a non-negative integer")
	}
	deployments, err := handler.listDeploymentsByApplication.Execute(
		ctx.UserContext(),
		applicationID,
		limit,
		offset,
	)
	if errors.Is(err, applicationdomain.ErrNotFound) {
		return httpx.WriteError(ctx, fiber.StatusNotFound, "application_not_found", "Application not found")
	}
	if err != nil {
		return httpx.WriteInternalError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(listResponse{
		Deployments: deployments,
		Limit:       limit,
		Offset:      offset,
	})
}

func parsePositiveID(raw, resource string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New(resource + " ID must be a positive integer")
	}
	return id, nil
}

func parseQueryInteger(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	return value, err
}
