package usecase

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/deployments"
	"github.com/irvaniamirali/serve-service/internal/deployments/ports"
)

// ListDeploymentsByApplication returns one page of an application's history.
type ListDeploymentsByApplication struct {
	repository        ports.Repository
	applicationReader applicationReader
}

func NewListDeploymentsByApplication(
	repository ports.Repository,
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
) ([]deployments.Deployment, error) {
	if applicationID < 1 || limit < 1 || limit > deployments.MaxPageSize || offset < 0 {
		return nil, deployments.ErrValidation
	}
	if _, err := usecase.applicationReader.GetApplicationByID(ctx, applicationID); err != nil {
		return nil, err
	}

	return usecase.repository.ListByApplication(ctx, applicationID, limit, offset)
}
