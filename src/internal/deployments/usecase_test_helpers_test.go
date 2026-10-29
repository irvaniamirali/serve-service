package deployments

import (
	"context"
	"errors"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

type applicationReaderFake struct {
	application applications.Application
	missing     bool
	err         error
	calls       int
}

func (reader *applicationReaderFake) GetApplicationByID(
	_ context.Context,
	id int64,
) (applications.Application, error) {
	reader.calls++
	if reader.err != nil {
		return applications.Application{}, reader.err
	}
	if reader.missing || id != 1 {
		return applications.Application{}, applications.ErrNotFound
	}
	if reader.application.ID == 0 {
		reader.application = applications.Application{
			ID:            id,
			RepositoryURL: "https://github.com/acme/demo.git",
			Branch:        "configured-branch",
		}
	}
	return reader.application, nil
}

type defaultCommitResolverFake struct {
	sha   string
	err   error
	calls int
}

func (resolver *defaultCommitResolverFake) ResolveDefaultBranchHead(
	_ context.Context,
	_ applications.Application,
) (string, error) {
	resolver.calls++
	return resolver.sha, resolver.err
}

type deploymentRepositoryFake struct {
	deployment    Deployment
	commitSHA     *string
	createErr     error
	createCalls   int
	getErr        error
	getCalls      int
	listErr       error
	listCalls     int
	listed        []Deployment
	listedAppID   int64
	listedLimit   int
	listedOffset  int
	transitionErr error
}

func (repository *deploymentRepositoryFake) Create(
	_ context.Context,
	applicationID int64,
	commitSHA *string,
) (Deployment, error) {
	repository.createCalls++
	if repository.createErr != nil {
		return Deployment{}, repository.createErr
	}
	repository.commitSHA = commitSHA
	repository.deployment = Deployment{
		ID:            1,
		ApplicationID: applicationID,
		Status:        StatusQueued,
		CommitSHA:     commitSHA,
	}
	return repository.deployment, nil
}

func (repository *deploymentRepositoryFake) GetByID(
	_ context.Context,
	id int64,
) (Deployment, error) {
	repository.getCalls++
	if repository.getErr != nil {
		return Deployment{}, repository.getErr
	}
	if repository.deployment.ID != id {
		return Deployment{}, ErrNotFound
	}
	return repository.deployment, nil
}

func (repository *deploymentRepositoryFake) ListByApplication(
	_ context.Context,
	applicationID int64,
	limit int,
	offset int,
) ([]Deployment, error) {
	repository.listCalls++
	repository.listedAppID = applicationID
	repository.listedLimit = limit
	repository.listedOffset = offset
	return repository.listed, repository.listErr
}

func (repository *deploymentRepositoryFake) TransitionStatus(
	_ context.Context,
	id int64,
	expectedStatus string,
	nextStatus string,
	failureMessage *string,
) (Deployment, error) {
	if repository.transitionErr != nil {
		return Deployment{}, repository.transitionErr
	}
	if repository.deployment.ID != id {
		return Deployment{}, ErrNotFound
	}
	if repository.deployment.Status != expectedStatus {
		return Deployment{}, ErrConcurrentTransition
	}
	repository.deployment.Status = nextStatus
	repository.deployment.ErrorMessage = failureMessage
	return repository.deployment, nil
}

var errRepositoryFailure = errors.New("repository failure")
