package usecase

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/domains/deployments"
	"github.com/irvaniamirali/serve-service/internal/domains/deployments/ports"
)

// GetDeploymentByID retrieves a deployment by its identifier.
type GetDeploymentByID struct {
	repository ports.Repository
}

func NewGetDeploymentByID(repository ports.Repository) *GetDeploymentByID {
	return &GetDeploymentByID{repository: repository}
}

func (usecase *GetDeploymentByID) Execute(
	ctx context.Context,
	id int64,
) (deployments.Deployment, error) {
	if id < 1 {
		return deployments.Deployment{}, deployments.ErrNotFound
	}
	return usecase.repository.GetByID(ctx, id)
}
