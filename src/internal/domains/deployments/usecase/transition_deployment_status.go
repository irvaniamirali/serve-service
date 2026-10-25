package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/irvaniamirali/serve-service/internal/domains/deployments"
	"github.com/irvaniamirali/serve-service/internal/domains/deployments/ports"
)

const maxFailureMessageSize = 2000

// TransitionDeploymentStatus validates and persists a lifecycle transition.
type TransitionDeploymentStatus struct {
	repository ports.Repository
}

func NewTransitionDeploymentStatus(
	repository ports.Repository,
) *TransitionDeploymentStatus {
	return &TransitionDeploymentStatus{repository: repository}
}

func (usecase *TransitionDeploymentStatus) Execute(
	ctx context.Context,
	id int64,
	nextStatus string,
	failureMessage string,
) (deployments.Deployment, error) {
	if id < 1 {
		return deployments.Deployment{}, deployments.ErrNotFound
	}

	failureMessage = strings.TrimSpace(failureMessage)
	if nextStatus == deployments.StatusFailed {
		if failureMessage == "" || len([]rune(failureMessage)) > maxFailureMessageSize {
			return deployments.Deployment{}, fmt.Errorf(
				"%w: a failure message of at most %d characters is required",
				deployments.ErrValidation,
				maxFailureMessageSize,
			)
		}
	} else if failureMessage != "" {
		return deployments.Deployment{}, fmt.Errorf(
			"%w: failure message is only valid when transitioning to failed",
			deployments.ErrValidation,
		)
	}

	current, err := usecase.repository.GetByID(ctx, id)
	if err != nil {
		return deployments.Deployment{}, err
	}
	if !deployments.CanTransition(current.Status, nextStatus) {
		return deployments.Deployment{}, fmt.Errorf(
			"%w: %s to %s",
			deployments.ErrInvalidTransition,
			current.Status,
			nextStatus,
		)
	}

	var message *string
	if failureMessage != "" {
		message = &failureMessage
	}
	deployment, err := usecase.repository.TransitionStatus(
		ctx,
		id,
		current.Status,
		nextStatus,
		message,
	)
	if errors.Is(err, deployments.ErrConcurrentTransition) {
		return deployments.Deployment{}, fmt.Errorf(
			"%w: deployment status changed concurrently",
			deployments.ErrInvalidTransition,
		)
	}
	return deployment, err
}
