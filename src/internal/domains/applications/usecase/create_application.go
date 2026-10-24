package usecase

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/domains/applications"
	"github.com/irvaniamirali/serve-service/internal/domains/applications/ports"
)

// CreateApplicationInput contains the data required to create an application.
type CreateApplicationInput struct {
	Name               string
	RepositoryURL      string
	Branch             string
	GitHubRepositoryID *int64
	WebhookSecret      string
}

// CreateApplication handles the application creation use case.
type CreateApplication struct {
	repository ports.Repository
}

// NewCreateApplication creates the application use case.
func NewCreateApplication(
	repository ports.Repository,
) *CreateApplication {
	return &CreateApplication{repository: repository}
}

// Execute validates and persists a new application.
func (usecase *CreateApplication) Execute(
	ctx context.Context,
	input CreateApplicationInput,
) (applications.Application, error) {
	application := applications.Application{
		Name:               input.Name,
		RepositoryURL:      input.RepositoryURL,
		Branch:             input.Branch,
		GitHubRepositoryID: input.GitHubRepositoryID,
		WebhookSecret:      input.WebhookSecret,
	}

	if err := applications.Validate(application); err != nil {
		return applications.Application{}, err
	}

	return usecase.repository.CreateApplication(ctx, application)
}
