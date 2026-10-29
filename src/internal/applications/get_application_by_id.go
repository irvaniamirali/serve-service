package applications

import (
	"context"
)

// GetApplicationByID handles retrieving an application by its identifier.
type GetApplicationByID struct {
	repository ApplicationRepositoryInterface
}

// NewGetApplicationByID creates the application retrieval use case.
func NewGetApplicationByID(
	repository ApplicationRepositoryInterface,
) *GetApplicationByID {
	return &GetApplicationByID{repository: repository}
}

// Execute retrieves an application by its stable identifier.
func (usecase *GetApplicationByID) Execute(
	ctx context.Context,
	id int64,
) (Application, error) {
	if id < 1 {
		return Application{}, ErrNotFound
	}

	return usecase.repository.GetApplicationByID(ctx, id)
}
