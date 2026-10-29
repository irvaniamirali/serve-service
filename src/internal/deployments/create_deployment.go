package deployments

import (
	"context"
	"fmt"
	"strings"

	"github.com/irvaniamirali/serve-service/internal/domains/applications"
	"github.com/irvaniamirali/serve-service/internal/github"
)

// CreateDeployment validates and persists a queued deployment request.
type CreateDeployment struct {
	repository        Repository
	applicationReader applicationReader
	commitResolver    DefaultCommitResolver
}

func NewCreateDeployment(
	repository Repository,
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
) (Deployment, error) {
	if applicationID < 1 {
		return Deployment{}, ErrValidation
	}
	application, err := usecase.applicationReader.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return Deployment{}, err
	}

	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA != "" && !github.IsDeployableCommitSHA(commitSHA) {
		return Deployment{}, fmt.Errorf(
			"%w: commit_sha must be a 40- or 64-character hexadecimal Git object ID",
			ErrValidation,
		)
	}
	if commitSHA == "" {
		if usecase.commitResolver == nil {
			return Deployment{}, fmt.Errorf(
				"%w: default-branch resolver is not configured",
				ErrCommitResolution,
			)
		}
		commitSHA, err = usecase.commitResolver.ResolveDefaultBranchHead(ctx, application)
		if err != nil {
			return Deployment{}, fmt.Errorf("%w: %w", ErrCommitResolution, err)
		}
		if !github.IsDeployableCommitSHA(commitSHA) {
			return Deployment{}, fmt.Errorf(
				"%w: GitHub returned an invalid commit SHA",
				ErrCommitResolution,
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
