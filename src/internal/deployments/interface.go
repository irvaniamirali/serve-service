package deployments

import (
	"context"
)

// DeploymentRepositoryInterface defines persistence operations required by deployment use cases.
type DeploymentRepositoryInterface interface {
	Create(context.Context, int64, *string) (Deployment, error)
	GetByID(context.Context, int64) (Deployment, error)
	ListByApplication(context.Context, int64, int, int) ([]Deployment, error)
	TransitionStatus(context.Context, int64, string, string, *string) (Deployment, error)
}
