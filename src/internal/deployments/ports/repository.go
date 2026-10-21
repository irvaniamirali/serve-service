package ports

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/deployments"
)

// Repository defines persistence operations required by deployment use cases.
type Repository interface {
	Create(context.Context, int64, *string) (deployments.Deployment, error)
	GetByID(context.Context, int64) (deployments.Deployment, error)
	ListByApplication(context.Context, int64, int, int) ([]deployments.Deployment, error)
	TransitionStatus(context.Context, int64, string, string, *string) (deployments.Deployment, error)
}
