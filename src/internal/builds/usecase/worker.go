package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/irvaniamirali/serve-service/internal/applications"
	applicationports "github.com/irvaniamirali/serve-service/internal/applications/ports"
	"github.com/irvaniamirali/serve-service/internal/deployments"
	"github.com/irvaniamirali/serve-service/internal/github"
)

const (
	buildPollInterval     = 2 * time.Second
	maxClaimRetryInterval = 30 * time.Second
	buildTimeout          = 20 * time.Minute
	resultSaveTimeout     = 10 * time.Second
)

// BuildResult records the local image identity and bounded diagnostics from one build attempt.
type BuildResult struct {
	ImageReference string
	ImageDigest    string
	Log            string
}

type buildQueue interface {
	ClaimNextQueued(context.Context) (deployments.Deployment, error)
	RecordBuildResult(
		context.Context,
		int64,
		bool,
		*string,
		*string,
		string,
		*string,
	) (deployments.Deployment, error)
}

type builder interface {
	Build(context.Context, applications.Application, string) (BuildResult, error)
}

type Worker struct {
	deployments  buildQueue
	applications applicationports.Repository
	builder      builder
	logger       *slog.Logger
	pollInterval time.Duration
	maxRetryWait time.Duration
}

// NewWorker wires the durable queue, application lookup, and image builder for background execution.
func NewWorker(
	deploymentRepository buildQueue,
	applicationRepository applicationports.Repository,
	imageBuilder builder,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		deployments:  deploymentRepository,
		applications: applicationRepository,
		builder:      imageBuilder,
		logger:       logger,
		pollInterval: buildPollInterval,
		maxRetryWait: maxClaimRetryInterval,
	}
}

// Run polls the durable deployment queue and builds one deployment at a time.
func (worker *Worker) Run(ctx context.Context) error {
	if worker.logger == nil {
		return fmt.Errorf("build worker logger is required")
	}

	retryWait := worker.pollInterval
	if retryWait <= 0 {
		retryWait = buildPollInterval
	}
	maxRetryWait := worker.maxRetryWait
	if maxRetryWait < retryWait {
		maxRetryWait = retryWait
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		deployment, err := worker.deployments.ClaimNextQueued(ctx)
		if errors.Is(err, deployments.ErrNoQueuedBuild) {
			retryWait = worker.pollInterval
			if retryWait <= 0 {
				retryWait = buildPollInterval
			}
			if !waitForPoll(ctx, retryWait) {
				return nil
			}
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			worker.logger.Error("claim queued deployment build", "error", err)
			if !waitForPoll(ctx, retryWait) {
				return nil
			}
			retryWait *= 2
			if retryWait > maxRetryWait {
				retryWait = maxRetryWait
			}
			continue
		}

		retryWait = worker.pollInterval
		if retryWait <= 0 {
			retryWait = buildPollInterval
		}
		if err := worker.runBuild(ctx, deployment); err != nil {
			worker.logger.Error(
				"process deployment build",
				"deployment_id", deployment.ID,
				"error", err,
			)
		}
	}
}

func waitForPoll(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (worker *Worker) runBuild(ctx context.Context, deployment deployments.Deployment) error {
	if deployment.Status != deployments.StatusBuilding {
		return fmt.Errorf("deployment %d is not in building state", deployment.ID)
	}

	buildCtx, cancel := context.WithTimeout(ctx, buildTimeout)
	defer cancel()

	result, buildErr := worker.buildDeployment(buildCtx, deployment)
	if buildErr == nil && (strings.TrimSpace(result.ImageReference) == "" || strings.TrimSpace(result.ImageDigest) == "") {
		buildErr = fmt.Errorf("image builder returned an incomplete image identity")
	}
	var imageReference, imageDigest, failureMessage *string
	if buildErr != nil {
		result.Log = appendBuildLog(result.Log, "Build failed: "+buildErr.Error())
		message := safeFailureMessage(buildErr)
		failureMessage = &message
	} else {
		imageReference = &result.ImageReference
		imageDigest = &result.ImageDigest
	}

	saveCtx, saveCancel := context.WithTimeout(context.Background(), resultSaveTimeout)
	defer saveCancel()
	_, err := worker.deployments.RecordBuildResult(
		saveCtx,
		deployment.ID,
		buildErr == nil,
		imageReference,
		imageDigest,
		result.Log,
		failureMessage,
	)
	if err != nil {
		return fmt.Errorf("persist build result for deployment %d: %w", deployment.ID, err)
	}
	if buildErr != nil {
		worker.logger.Warn(
			"deployment build failed",
			"deployment_id", deployment.ID,
			"error", *failureMessage,
		)
		return nil
	}

	worker.logger.Info(
		"deployment build completed",
		"deployment_id", deployment.ID,
		"image_reference", result.ImageReference,
		"image_digest", result.ImageDigest,
	)
	return nil
}

func (worker *Worker) buildDeployment(
	ctx context.Context,
	deployment deployments.Deployment,
) (BuildResult, error) {
	if deployment.CommitSHA == nil || !github.IsDeployableCommitSHA(*deployment.CommitSHA) {
		return BuildResult{}, fmt.Errorf("deployment %d has no valid full commit SHA", deployment.ID)
	}

	application, err := worker.applications.GetApplicationByID(ctx, deployment.ApplicationID)
	if err != nil {
		return BuildResult{}, fmt.Errorf("load application %d for deployment %d: %w", deployment.ApplicationID, deployment.ID, err)
	}
	if strings.TrimSpace(application.RepositoryURL) == "" {
		return BuildResult{}, fmt.Errorf("application %d has no repository clone URL", application.ID)
	}

	return worker.builder.Build(ctx, application, *deployment.CommitSHA)
}

func safeFailureMessage(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "build timed out"
	case errors.Is(err, context.Canceled):
		return "build canceled"
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "no valid full commit sha"):
		return "deployment has no valid full commit SHA"
	case strings.Contains(message, "repository clone url"):
		return "application repository clone URL is missing or unsupported"
	case strings.Contains(message, "load application"):
		return "registered application could not be loaded"
	case strings.Contains(message, "root-level dockerfile"):
		return "repository must contain a readable root-level Dockerfile"
	case strings.Contains(message, "unsupported symlink"):
		return "repository build context contains unsupported symlinks"
	case strings.Contains(message, "clone repository"):
		return "repository clone failed; see build log for details"
	case strings.Contains(message, "fetch requested commit"):
		return "requested commit could not be fetched; see build log for details"
	case strings.Contains(message, "check out requested commit"):
		return "requested commit could not be checked out; see build log for details"
	case strings.Contains(message, "does not match requested commit"):
		return "checked-out commit did not match the requested commit"
	case strings.Contains(message, "verify checked-out commit"):
		return "checked-out commit could not be verified; see build log for details"
	case strings.Contains(message, "docker"), strings.Contains(message, "image"):
		return "Docker image build or inspection failed; see build log for details"
	case strings.Contains(message, "temporary build workspace"):
		return "temporary build workspace could not be prepared"
	case strings.Contains(message, "incomplete image identity"):
		return "image builder returned an incomplete image identity"
	default:
		return "build failed; see persisted build log for details"
	}
}

func appendBuildLog(current, line string) string {
	if current == "" {
		return line
	}
	return current + "\n" + line
}
