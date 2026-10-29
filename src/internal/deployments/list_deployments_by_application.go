package deployments

import (
	"context"
)

// ListDeploymentsByApplication returns one page of an application's history.
type ListDeploymentsByApplication struct {
	repository        DeploymentRepositoryInterface
	applicationReader applicationReader
}

func NewListDeploymentsByApplication(
	repository DeploymentRepositoryInterface,
	applicationReader applicationReader,
) *ListDeploymentsByApplication {
	return &ListDeploymentsByApplication{
		repository:        repository,
		applicationReader: applicationReader,
	}
}

func (usecase *ListDeploymentsByApplication) Execute(
	ctx context.Context,
	applicationID int64,
	limit int,
	offset int,
) ([]Deployment, error) {
	if applicationID < 1 || limit < 1 || limit > MaxPageSize || offset < 0 {
		return nil, ErrValidation
	}
	if _, err := usecase.applicationReader.GetApplicationByID(ctx, applicationID); err != nil {
		return nil, err
	}

	return usecase.repository.ListByApplication(ctx, applicationID, limit, offset)
}
