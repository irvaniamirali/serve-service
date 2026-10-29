package deployments

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maxFailureMessageSize = 2000

// TransitionDeploymentStatus validates and persists a lifecycle transition.
type TransitionDeploymentStatus struct {
	repository Repository
}

func NewTransitionDeploymentStatus(
	repository Repository,
) *TransitionDeploymentStatus {
	return &TransitionDeploymentStatus{repository: repository}
}

func (usecase *TransitionDeploymentStatus) Execute(
	ctx context.Context,
	id int64,
	nextStatus string,
	failureMessage string,
) (Deployment, error) {
	if id < 1 {
		return Deployment{}, ErrNotFound
	}

	failureMessage = strings.TrimSpace(failureMessage)
	if nextStatus == StatusFailed {
		if failureMessage == "" || len([]rune(failureMessage)) > maxFailureMessageSize {
			return Deployment{}, fmt.Errorf(
				"%w: a failure message of at most %d characters is required",
				ErrValidation,
				maxFailureMessageSize,
			)
		}
	} else if failureMessage != "" {
		return Deployment{}, fmt.Errorf(
			"%w: failure message is only valid when transitioning to failed",
			ErrValidation,
		)
	}

	current, err := usecase.repository.GetByID(ctx, id)
	if err != nil {
		return Deployment{}, err
	}
	if !CanTransition(current.Status, nextStatus) {
		return Deployment{}, fmt.Errorf(
			"%w: %s to %s",
			ErrInvalidTransition,
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
	if errors.Is(err, ErrConcurrentTransition) {
		return Deployment{}, fmt.Errorf(
			"%w: deployment status changed concurrently",
			ErrInvalidTransition,
		)
	}
	return deployment, err
}
