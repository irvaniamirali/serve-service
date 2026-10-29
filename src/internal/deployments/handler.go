package deployments

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/infrastructure/httpx"
)

// Handler handles HTTP requests related to deployments.
type Handler struct {
	createDeployment             *CreateDeployment
	getDeploymentByID            *GetDeploymentByID
	listDeploymentsByApplication *ListDeploymentsByApplication
}

// NewHandler creates a deployment handler with its required use cases.
func NewHandler(
	createDeployment *CreateDeployment,
	getDeploymentByID *GetDeploymentByID,
	listDeploymentsByApplication *ListDeploymentsByApplication,
) *Handler {
	return &Handler{
		createDeployment:             createDeployment,
		getDeploymentByID:            getDeploymentByID,
		listDeploymentsByApplication: listDeploymentsByApplication,
	}
}

// RegisterRoutes registers the deployment HTTP endpoints.
func (handler *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/applications/:id/deployments", handler.create)
	router.Get("/applications/:id/deployments", handler.listByApplication)
	router.Get("/deployments/:id", handler.get)
	router.Get("/deployments/:id/logs", handler.getBuildLog)
}

type CreateDeploymentRequest struct {
	CommitSHA string `json:"commit_sha"`
}

type ListDeploymentsResponse struct {
	Deployments []Deployment `json:"deployments"`
	Limit       int          `json:"limit"`
	Offset      int          `json:"offset"`
}

type BuildLogResponse struct {
	DeploymentID int64  `json:"deployment_id"`
	Log          string `json:"log"`
}

func (handler *Handler) create(ctx *fiber.Ctx) error {
	applicationID, err := parsePositiveID(ctx.Params("id"), "Application")
	if err != nil {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			err.Error(),
		)
	}

	var request CreateDeploymentRequest

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
	if errors.Is(err, ErrValidation) {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"validation_error",
			err.Error(),
		)
	}

	if errors.Is(err, applications.ErrNotFound) {
		return httpx.WriteError(
			ctx,
			fiber.StatusNotFound,
			"application_not_found",
			"Application not found",
		)
	}

	if errors.Is(err, ErrCommitResolution) {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadGateway,
			"commit_resolution_failed",
			"Could not resolve the latest commit on the repository's GitHub default branch. Check repository metadata and GitHub API access.",
		)
	}

	if err != nil {
		return httpx.WriteInternalError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(deployment)
}

func (handler *Handler) get(ctx *fiber.Ctx) error {
	deploymentID, err := parsePositiveID(ctx.Params("id"), "Deployment")
	if err != nil {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			err.Error(),
		)
	}

	deployment, err := handler.getDeploymentByID.Execute(
		ctx.UserContext(),
		deploymentID,
	)
	if err != nil {
		return writeDeploymentReadError(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(deployment)
}

func (handler *Handler) getBuildLog(ctx *fiber.Ctx) error {
	deploymentID, err := parsePositiveID(ctx.Params("id"), "Deployment")
	if err != nil {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			err.Error(),
		)
	}

	deployment, err := handler.getDeploymentByID.Execute(
		ctx.UserContext(),
		deploymentID,
	)
	if err != nil {
		return writeDeploymentReadError(ctx, err)
	}

	ctx.Set(fiber.HeaderCacheControl, "no-store")

	buildLog := ""
	if deployment.BuildLog != nil {
		buildLog = *deployment.BuildLog
	}

	return ctx.Status(fiber.StatusOK).JSON(BuildLogResponse{
		DeploymentID: deployment.ID,
		Log:          buildLog,
	})
}

func (handler *Handler) listByApplication(ctx *fiber.Ctx) error {
	applicationID, err := parsePositiveID(ctx.Params("id"), "Application")
	if err != nil {
		return httpx.WriteError(
			ctx,
			fiber.StatusBadRequest,
			"invalid_request",
			err.Error(),
		)
	}

	limit, err := parseQueryInteger(
		ctx.Query("limit"),
		DefaultPageSize,
	)
	if err != nil || limit < 1 || limit > MaxPageSize {
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

	deploymentList, err := handler.listDeploymentsByApplication.Execute(
		ctx.UserContext(),
		applicationID,
		limit,
		offset,
	)
	if errors.Is(err, applications.ErrNotFound) {
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

	return ctx.Status(fiber.StatusOK).JSON(ListDeploymentsResponse{
		Deployments: deploymentList,
		Limit:       limit,
		Offset:      offset,
	})
}

// writeDeploymentReadError translates deployment lookup errors into HTTP responses.
func writeDeploymentReadError(ctx *fiber.Ctx, err error) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.WriteError(
			ctx,
			fiber.StatusNotFound,
			"deployment_not_found",
			"Deployment not found",
		)
	}

	return httpx.WriteInternalError(ctx, err)
}

// parsePositiveID parses a resource ID that must be a positive integer.
func parsePositiveID(rawID string, resourceName string) (int64, error) {
	parsedID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || parsedID < 1 {
		return 0, errors.New(resourceName + " ID must be a positive integer")
	}

	return parsedID, nil
}

// parseQueryInteger parses an integer query parameter or returns its fallback.
func parseQueryInteger(rawValue string, fallback int) (int, error) {
	if rawValue == "" {
		return fallback, nil
	}

	return strconv.Atoi(rawValue)
}
