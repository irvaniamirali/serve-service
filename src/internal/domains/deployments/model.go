package deployments

import (
	"errors"
	"time"
)

const (
	StatusQueued    = "queued"
	StatusBuilding  = "building"
	StatusBuilt     = "built"
	StatusDeploying = "deploying"
	StatusHealthy   = "healthy"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"

	DefaultPageSize = 20
	MaxPageSize     = 100
)

var (
	ErrValidation           = errors.New("deployment validation failed")
	ErrNotFound             = errors.New("deployment not found")
	ErrInvalidTransition    = errors.New("invalid deployment status transition")
	ErrConcurrentTransition = errors.New("deployment status changed concurrently")
	ErrNoQueuedBuild        = errors.New("no queued deployment build")
	ErrCommitResolution     = errors.New("could not resolve deployment commit")
)

// Deployment records one attempt to deploy an application revision.
type Deployment struct {
	ID              int64      `json:"id"`
	ApplicationID   int64      `json:"application_id"`
	Status          string     `json:"status"`
	CommitSHA       *string    `json:"commit_sha,omitempty"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	ImageReference  *string    `json:"image_reference,omitempty"`
	ImageDigest     *string    `json:"image_digest,omitempty"`
	BuildStartedAt  *time.Time `json:"build_started_at,omitempty"`
	BuildFinishedAt *time.Time `json:"build_finished_at,omitempty"`
	BuildLog        *string    `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CanTransition reports whether the lifecycle permits moving between states.
func CanTransition(from, to string) bool {
	switch from {
	case StatusQueued:
		return to == StatusBuilding || to == StatusCancelled
	case StatusBuilding:
		return to == StatusBuilt || to == StatusDeploying || to == StatusFailed || to == StatusCancelled
	case StatusBuilt:
		return to == StatusDeploying || to == StatusFailed || to == StatusCancelled
	case StatusDeploying:
		return to == StatusHealthy || to == StatusFailed || to == StatusCancelled
	default:
		return false
	}
}
