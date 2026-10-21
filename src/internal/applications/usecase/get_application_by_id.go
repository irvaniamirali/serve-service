package usecase

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

// GetApplicationByID handles retrieving an application by its identifier.
type GetApplicationByID struct {
	repository applications.Repository
}

// NewGetApplicationByID creates the application retrieval use case.
func NewGetApplicationByID(
	repository applications.Repository,
) *GetApplicationByID {
	return &GetApplicationByID{repository: repository}
}

// Execute retrieves an application by its stable identifier.
func (usecase *GetApplicationByID) Execute(
	ctx context.Context,
	id int64,
) (applications.Application, error) {
	if id < 1 {
		return applications.Application{}, applications.ErrNotFound
	}

	return usecase.repository.GetApplicationByID(ctx, id)
}
