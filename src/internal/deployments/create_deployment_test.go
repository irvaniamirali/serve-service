package deployments

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

func TestCreateDeploymentValidatesAndNormalizesCommit(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	resolver := &defaultCommitResolverFake{}
	usecase := NewCreateDeployment(repository, &applicationReaderFake{}, resolver)

	created, err := usecase.Execute(
		context.Background(),
		1,
		" ABCDEF0123456789ABCDEF0123456789ABCDEF01 ",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if created.Status != StatusQueued ||
		repository.commitSHA == nil ||
		*repository.commitSHA != "abcdef0123456789abcdef0123456789abcdef01" {
		t.Fatalf("created deployment = %+v, commit SHA = %v", created, repository.commitSHA)
	}

	_, err = usecase.Execute(context.Background(), 1, "not-a-commit")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid commit error = %v, want ErrValidation", err)
	}
	if repository.createCalls != 1 {
		t.Fatalf("Create calls = %d, want 1", repository.createCalls)
	}
	if resolver.calls != 0 {
		t.Fatalf("default commit resolver calls = %d, want 0 for explicit commits", resolver.calls)
	}
}

func TestCreateDeploymentResolvesAndPersistsDefaultBranchHead(t *testing.T) {
	const resolvedSHA = "abcdef0123456789abcdef0123456789abcdef01"
	const laterBranchSHA = "0123456789abcdef0123456789abcdef01234567"
	repository := &deploymentRepositoryFake{}
	resolver := &defaultCommitResolverFake{sha: "ABCDEF0123456789ABCDEF0123456789ABCDEF01"}
	usecase := NewCreateDeployment(repository, &applicationReaderFake{}, resolver)

	created, err := usecase.Execute(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("default commit resolver calls = %d, want 1", resolver.calls)
	}
	if repository.commitSHA == nil || *repository.commitSHA != resolvedSHA ||
		created.CommitSHA == nil || *created.CommitSHA != resolvedSHA {
		t.Fatalf("persisted commit SHA = %v and response SHA = %v, want %s", repository.commitSHA, created.CommitSHA, resolvedSHA)
	}

	resolver.sha = laterBranchSHA
	if repository.deployment.CommitSHA == nil || *repository.deployment.CommitSHA != resolvedSHA {
		t.Fatalf("queued deployment SHA changed after branch advanced: %v", repository.deployment.CommitSHA)
	}
}

func TestCreateDeploymentDoesNotFallbackWhenExplicitSHAIsInvalid(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	resolver := &defaultCommitResolverFake{sha: "abcdef0123456789abcdef0123456789abcdef01"}
	usecase := NewCreateDeployment(repository, &applicationReaderFake{}, resolver)

	for _, invalidSHA := range []string{"invalid-sha", strings.Repeat("0", 40)} {
		_, err := usecase.Execute(context.Background(), 1, invalidSHA)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("Execute(%q) error = %v, want ErrValidation", invalidSHA, err)
		}
	}
	if repository.createCalls != 0 || resolver.calls != 0 {
		t.Fatalf("invalid SHA triggered fallback or persistence: resolver calls=%d, create calls=%d", resolver.calls, repository.createCalls)
	}
}

func TestCreateDeploymentFailsWhenDefaultBranchCannotBeResolved(t *testing.T) {
	resolutionError := errors.New("GitHub API returned HTTP 404")
	repository := &deploymentRepositoryFake{}
	resolver := &defaultCommitResolverFake{err: resolutionError}
	usecase := NewCreateDeployment(repository, &applicationReaderFake{}, resolver)

	_, err := usecase.Execute(context.Background(), 1, "")
	if !errors.Is(err, ErrCommitResolution) || !errors.Is(err, resolutionError) {
		t.Fatalf("Execute() error = %v, want wrapped commit-resolution error", err)
	}
	if repository.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0 after resolution failure", repository.createCalls)
	}
}

func TestCreateDeploymentRejectsInvalidResolvedSHA(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	resolver := &defaultCommitResolverFake{sha: "branch-tip"}
	usecase := NewCreateDeployment(repository, &applicationReaderFake{}, resolver)

	_, err := usecase.Execute(context.Background(), 1, "")
	if !errors.Is(err, ErrCommitResolution) {
		t.Fatalf("Execute() error = %v, want ErrCommitResolution", err)
	}
	if repository.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0 for invalid resolved SHA", repository.createCalls)
	}
}

func TestCreateDeploymentRequiresExistingApplication(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	reader := &applicationReaderFake{missing: true}
	usecase := NewCreateDeployment(repository, reader, &defaultCommitResolverFake{})

	_, err := usecase.Execute(context.Background(), 1, "")
	if !errors.Is(err, applications.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want applications.ErrNotFound", err)
	}
	if repository.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", repository.createCalls)
	}
}

func TestCreateDeploymentReturnsPersistenceError(t *testing.T) {
	repository := &deploymentRepositoryFake{createErr: errRepositoryFailure}
	usecase := NewCreateDeployment(
		repository,
		&applicationReaderFake{},
		&defaultCommitResolverFake{},
	)

	_, err := usecase.Execute(context.Background(), 1, "abcdef0123456789abcdef0123456789abcdef01")
	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("Execute() error = %v, want wrapped repository error", err)
	}
}
