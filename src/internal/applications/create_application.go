package applications

import (
	"context"
)

// CreateApplicationParams contains the parameters required to create an application.
type CreateApplicationParams struct {
	Name               string
	RepositoryURL      string
	Branch             string
	GitHubRepositoryID *int64
	WebhookSecret      string
}

// CreateApplication handles the application creation use case.
type CreateApplication struct {
	repository ApplicationRepositoryInterface
}

// NewCreateApplication creates the application use case.
func NewCreateApplication(
	repository ApplicationRepositoryInterface,
) *CreateApplication {
	return &CreateApplication{repository: repository}
}

// Execute validates and persists a new application.
func (usecase *CreateApplication) Execute(
	ctx context.Context,
	params CreateApplicationParams,
) (Application, error) {
	application := Application{
		Name:               params.Name,
		RepositoryURL:      params.RepositoryURL,
		Branch:             params.Branch,
		GitHubRepositoryID: params.GitHubRepositoryID,
		WebhookSecret:      params.WebhookSecret,
	}

	if err := Validate(application); err != nil {
		return Application{}, err
	}

	return usecase.repository.CreateApplication(ctx, application)
}
