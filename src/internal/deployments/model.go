package deployments

import (
	"errors"
	"time"
)

const (
	StatusQueued    = "queued"
	StatusBuilding  = "building"
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
)

// Deployment records one attempt to deploy an application revision.
type Deployment struct {
	ID            int64     `json:"id"`
	ApplicationID int64     `json:"application_id"`
	Status        string    `json:"status"`
	CommitSHA     *string   `json:"commit_sha,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DeploymentModel maps a deployment to the existing database schema.
type DeploymentModel struct {
	ID            int64     `gorm:"primaryKey;column:id"`
	ApplicationID int64     `gorm:"column:application_id"`
	Status        string    `gorm:"column:status;default:queued"`
	CommitSHA     *string   `gorm:"column:commit_sha"`
	ErrorMessage  *string   `gorm:"column:error_message"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (DeploymentModel) TableName() string {
	return "deployments"
}

// CanTransition reports whether the lifecycle permits moving between states.
func CanTransition(from, to string) bool {
	switch from {
	case StatusQueued:
		return to == StatusBuilding || to == StatusCancelled
	case StatusBuilding:
		return to == StatusDeploying || to == StatusFailed || to == StatusCancelled
	case StatusDeploying:
		return to == StatusHealthy || to == StatusFailed || to == StatusCancelled
	default:
		return false
	}
}
