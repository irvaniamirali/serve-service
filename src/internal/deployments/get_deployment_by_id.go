package deployments

import (
	"context"
)

// GetDeploymentByID retrieves a deployment by its identifier.
type GetDeploymentByID struct {
	repository Repository
}

func NewGetDeploymentByID(repository Repository) *GetDeploymentByID {
	return &GetDeploymentByID{repository: repository}
}

func (usecase *GetDeploymentByID) Execute(
	ctx context.Context,
	id int64,
) (Deployment, error) {
	if id < 1 {
		return Deployment{}, ErrNotFound
	}
	return usecase.repository.GetByID(ctx, id)
}
