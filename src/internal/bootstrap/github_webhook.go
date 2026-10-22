package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	applicationdomain "github.com/irvaniamirali/serve-service/internal/applications"
	applicationports "github.com/irvaniamirali/serve-service/internal/applications/ports"
	deploymentports "github.com/irvaniamirali/serve-service/internal/deployments/ports"
	githubpkg "github.com/irvaniamirali/serve-service/internal/github"
	"github.com/irvaniamirali/serve-service/internal/platform/httpx"
	"gorm.io/gorm"
)

var githubWebhookProcessor *GitHubWebhookProcessor

type GitHubWebhookProcessor struct {
	applicationRepository applicationports.Repository
	deploymentRepository  deploymentports.Repository
	secret                string
	db                    *gorm.DB
}

func setGitHubWebhookProcessor(
	applicationRepository applicationports.Repository,
	deploymentRepository deploymentports.Repository,
	secret string,
	db *gorm.DB,
) {
	githubWebhookProcessor = &GitHubWebhookProcessor{
		applicationRepository: applicationRepository,
		deploymentRepository:  deploymentRepository,
		secret:                secret,
		db:                    db,
	}
}

func registerGitHubWebhookRoute(app *fiber.App) {
	if githubWebhookProcessor == nil {
		return
	}
	app.Post("/api/v1/webhooks/github", githubWebhookProcessor.handle)
	app.Post("/api/v1/github/webhooks", githubWebhookProcessor.handle)
}

func (processor *GitHubWebhookProcessor) handle(ctx *fiber.Ctx) error {
	body := append([]byte(nil), ctx.Body()...)
	eventType := strings.TrimSpace(ctx.Get("X-GitHub-Event"))
	deliveryID := strings.TrimSpace(ctx.Get("X-GitHub-Delivery"))
	if eventType == "" {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_request", "Missing GitHub event type")
	}
	if deliveryID == "" {
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_request", "Missing GitHub delivery ID")
	}
	if err := githubpkg.VerifySignature(processor.secret, ctx.Get("X-Hub-Signature-256"), body); err != nil {
		if errors.Is(err, githubpkg.ErrMissingSecret) {
			return httpx.WriteError(ctx, fiber.StatusInternalServerError, "webhook_secret_missing", "GitHub webhook secret is not configured")
		}
		return httpx.WriteError(ctx, fiber.StatusUnauthorized, "invalid_signature", "Invalid GitHub webhook signature")
	}
	if processor.db != nil {
		if created, err := processor.beginDelivery(ctx.UserContext(), deliveryID, eventType); err != nil {
			return httpx.WriteInternalError(ctx, err)
		} else if !created {
			outcome, err := processor.getDeliveryOutcome(ctx.UserContext(), deliveryID)
			if err != nil {
				return httpx.WriteInternalError(ctx, err)
			}
			switch outcome {
			case "processed":
				return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "duplicate", "delivery_id": deliveryID})
			case "ignored":
				return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "ignored", "delivery_id": deliveryID})
			case "failed":
				return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "failed", "delivery_id": deliveryID})
			default:
				return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "duplicate", "delivery_id": deliveryID})
			}
		}
	}
	if eventType != "push" {
		_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, 0, 0, "ignored", nil, "event type ignored")
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "ignored", "event_type": eventType})
	}

	event, err := githubpkg.ParsePushEvent(body)
	if err != nil {
		_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, 0, 0, "failed", nil, err.Error())
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_payload", "Malformed GitHub push payload")
	}
	if !githubpkg.IsDeployableCommitSHA(event.After) {
		_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, 0, 0, "failed", nil, "missing or invalid commit SHA")
		return httpx.WriteError(ctx, fiber.StatusBadRequest, "invalid_payload", "Push payload must contain a valid commit SHA")
	}
	if !strings.HasPrefix(event.Ref, "refs/heads/") {
		_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, 0, 0, "ignored", nil, "branch deletion or non-branch ref")
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "ignored", "event_type": eventType, "ref": event.Ref})
	}

	application, err := processor.applicationRepository.GetApplicationByGitHubRepositoryID(ctx.UserContext(), event.Repository.ID)
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, 0, 0, "ignored", nil, "unknown repository")
			return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "ignored", "event_type": eventType})
		}
		_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, 0, 0, "failed", nil, err.Error())
		return httpx.WriteInternalError(ctx, err)
	}
	if !githubpkg.MatchesConfiguredBranch(application.Branch, event.Ref) {
		_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, application.ID, 0, "ignored", nil, "branch does not match configured deployment branch")
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "ignored", "branch": event.Ref})
	}

	deployment, err := processor.deploymentRepository.Create(ctx.UserContext(), application.ID, &event.After)
	if err != nil {
		_ = processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, application.ID, 0, "failed", nil, err.Error())
		return httpx.WriteInternalError(ctx, err)
	}
	if err := processor.updateDeliveryOutcome(ctx.UserContext(), deliveryID, eventType, application.ID, deployment.ID, "processed", &deployment.ID, ""); err != nil {
		slog.Error("record processed delivery", "delivery_id", deliveryID, "error", err)
	}
	return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "processed", "deployment_id": deployment.ID, "commit_sha": event.After})
}

func webhookSecretFromEnvironment() string {
	return strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_SECRET"))
}

func (processor *GitHubWebhookProcessor) beginDelivery(
	ctx context.Context,
	deliveryID string,
	eventType string,
) (bool, error) {
	if processor.db == nil {
		return true, nil
	}
	result := processor.db.WithContext(ctx).Exec(
		`INSERT INTO github_webhook_deliveries (delivery_id, event_type, application_id, deployment_id, outcome, failure_reason, created_at, updated_at)
		VALUES (?, ?, 0, 0, 'processing', '', NOW(), NOW())
		ON CONFLICT (delivery_id) DO NOTHING`,
		deliveryID,
		eventType,
	)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (processor *GitHubWebhookProcessor) getDeliveryOutcome(
	ctx context.Context,
	deliveryID string,
) (string, error) {
	if processor.db == nil {
		return "processed", nil
	}
	var outcome string
	err := processor.db.WithContext(ctx).Table("github_webhook_deliveries").Select("outcome").Where("delivery_id = ?", deliveryID).Scan(&outcome).Error
	if err != nil {
		return "", err
	}
	return outcome, nil
}

func (processor *GitHubWebhookProcessor) updateDeliveryOutcome(
	ctx context.Context,
	deliveryID string,
	eventType string,
	applicationID int64,
	deploymentID int64,
	outcome string,
	deploymentRef *int64,
	failureReason string,
) error {
	if processor.db == nil {
		return nil
	}
	if deploymentRef != nil {
		deploymentID = *deploymentRef
	}
	result := processor.db.WithContext(ctx).Exec(
		`UPDATE github_webhook_deliveries
		SET event_type = ?, application_id = ?, deployment_id = ?, outcome = ?, failure_reason = ?, updated_at = NOW()
		WHERE delivery_id = ?`,
		eventType,
		applicationID,
		deploymentID,
		outcome,
		failureReason,
		deliveryID,
	)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
