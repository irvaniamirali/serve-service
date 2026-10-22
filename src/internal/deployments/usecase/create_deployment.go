package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/deployments"
	"github.com/irvaniamirali/serve-service/internal/deployments/ports"
	"github.com/irvaniamirali/serve-service/internal/github"
)

// CreateDeployment validates and persists a queued deployment request.
type CreateDeployment struct {
	repository        ports.Repository
	applicationReader applicationReader
	commitResolver    DefaultCommitResolver
}

func NewCreateDeployment(
	repository ports.Repository,
	applicationReader applicationReader,
	commitResolver DefaultCommitResolver,
) *CreateDeployment {
	return &CreateDeployment{
		repository:        repository,
		applicationReader: applicationReader,
		commitResolver:    commitResolver,
	}
}

func (usecase *CreateDeployment) Execute(
	ctx context.Context,
	applicationID int64,
	commitSHA string,
) (deployments.Deployment, error) {
	if applicationID < 1 {
		return deployments.Deployment{}, deployments.ErrValidation
	}
	application, err := usecase.applicationReader.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return deployments.Deployment{}, err
	}

	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA != "" && !github.IsDeployableCommitSHA(commitSHA) {
		return deployments.Deployment{}, fmt.Errorf(
			"%w: commit_sha must be a 40- or 64-character hexadecimal Git object ID",
			deployments.ErrValidation,
		)
	}
	if commitSHA == "" {
		if usecase.commitResolver == nil {
			return deployments.Deployment{}, fmt.Errorf(
				"%w: default-branch resolver is not configured",
				deployments.ErrCommitResolution,
			)
		}
		commitSHA, err = usecase.commitResolver.ResolveDefaultBranchHead(ctx, application)
		if err != nil {
			return deployments.Deployment{}, fmt.Errorf("%w: %w", deployments.ErrCommitResolution, err)
		}
		if !github.IsDeployableCommitSHA(commitSHA) {
			return deployments.Deployment{}, fmt.Errorf(
				"%w: GitHub returned an invalid commit SHA",
				deployments.ErrCommitResolution,
			)
		}
	}

	commitSHA = strings.ToLower(commitSHA)
	return usecase.repository.Create(ctx, applicationID, &commitSHA)
}

// DefaultCommitResolver retrieves the HEAD commit for an application's GitHub default branch.
type DefaultCommitResolver interface {
	ResolveDefaultBranchHead(context.Context, applications.Application) (string, error)
}
