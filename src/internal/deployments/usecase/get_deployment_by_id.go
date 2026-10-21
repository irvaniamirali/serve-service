package usecase

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/deployments"
)

// GetDeploymentByID retrieves a deployment by its identifier.
type GetDeploymentByID struct {
	repository deployments.Repository
}

func NewGetDeploymentByID(repository deployments.Repository) *GetDeploymentByID {
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
