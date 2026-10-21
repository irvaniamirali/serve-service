package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/irvaniamirali/serve-service/internal/deployments"
)

// CreateDeployment validates and persists a queued deployment request.
type CreateDeployment struct {
	repository        deployments.Repository
	applicationReader applicationReader
}

func NewCreateDeployment(
	repository deployments.Repository,
	applicationReader applicationReader,
) *CreateDeployment {
	return &CreateDeployment{
		repository:        repository,
		applicationReader: applicationReader,
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
	if _, err := usecase.applicationReader.GetApplicationByID(ctx, applicationID); err != nil {
		return deployments.Deployment{}, err
	}

	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA != "" && !validCommitSHA(commitSHA) {
		return deployments.Deployment{}, fmt.Errorf(
			"%w: commit_sha must be a 40- or 64-character hexadecimal Git object ID",
			deployments.ErrValidation,
		)
	}
	if commitSHA == "" {
		return usecase.repository.Create(ctx, applicationID, nil)
	}

	commitSHA = strings.ToLower(commitSHA)
	return usecase.repository.Create(ctx, applicationID, &commitSHA)
}

func validCommitSHA(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') &&
			(character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}
