package usecase

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/applications/ports"
)

const (
	DefaultApplicationPageSize = 20
	MaxApplicationPageSize     = 100
)

// ListApplications retrieves a page of registered applications.
type ListApplications struct {
	repository ports.Repository
}

// NewListApplications creates the application listing use case.
func NewListApplications(repository ports.Repository) *ListApplications {
	return &ListApplications{repository: repository}
}

// Execute returns applications ordered by the repository's stable listing order.
func (usecase *ListApplications) Execute(
	ctx context.Context,
	limit int,
	offset int,
) ([]applications.Application, error) {
	if limit < 1 || limit > MaxApplicationPageSize || offset < 0 {
		return nil, applications.ErrValidation
	}

	return usecase.repository.ListApplications(ctx, limit, offset)
}
