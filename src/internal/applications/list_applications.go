package applications

import (
	"context"
)

const (
	DefaultApplicationPageSize = 20
	MaxApplicationPageSize     = 100
)

// ListApplications retrieves a page of registered applications.
type ListApplications struct {
	repository ApplicationRepositoryInterface
}

// NewListApplications creates the application listing use case.
func NewListApplications(repository ApplicationRepositoryInterface) *ListApplications {
	return &ListApplications{repository: repository}
}

// Execute returns applications ordered by the repository's stable listing order.
func (usecase *ListApplications) Execute(
	ctx context.Context,
	limit int,
	offset int,
) ([]Application, error) {
	if limit < 1 || limit > MaxApplicationPageSize || offset < 0 {
		return nil, ErrValidation
	}

	return usecase.repository.ListApplications(ctx, limit, offset)
}
