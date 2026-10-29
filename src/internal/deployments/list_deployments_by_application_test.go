package deployments

import (
	"context"
	"errors"
	"testing"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

func TestListDeploymentsByApplicationPassesValidatedPageToRepository(t *testing.T) {
	want := []Deployment{{ID: 7, ApplicationID: 2, Status: StatusQueued}}
	repository := &deploymentRepositoryFake{listed: want}
	reader := &applicationReaderFake{}

	result, err := NewListDeploymentsByApplication(repository, reader).Execute(
		context.Background(),
		1,
		25,
		50,
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result) != len(want) || result[0].ID != want[0].ID {
		t.Fatalf("Execute() = %+v, want %+v", result, want)
	}
	if reader.calls != 1 || repository.listCalls != 1 ||
		repository.listedAppID != 1 ||
		repository.listedLimit != 25 ||
		repository.listedOffset != 50 {
		t.Fatalf("reader calls=%d, repository calls=%d page=%d/%d/%d", reader.calls, repository.listCalls, repository.listedAppID, repository.listedLimit, repository.listedOffset)
	}
}

func TestListDeploymentsByApplicationRejectsInvalidPageWithoutRepositoryCall(t *testing.T) {
	tests := []struct {
		name        string
		application int64
		limit       int
		offset      int
	}{
		{name: "invalid application ID", application: 0, limit: 1},
		{name: "zero limit", application: 1},
		{name: "limit too large", application: 1, limit: MaxPageSize + 1},
		{name: "negative offset", application: 1, limit: 1, offset: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &deploymentRepositoryFake{}
			reader := &applicationReaderFake{}
			_, err := NewListDeploymentsByApplication(repository, reader).Execute(
				context.Background(),
				test.application,
				test.limit,
				test.offset,
			)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Execute() error = %v, want ErrValidation", err)
			}
			if repository.listCalls != 0 || reader.calls != 0 {
				t.Fatalf("invalid page triggered dependencies: repository calls=%d, reader calls=%d", repository.listCalls, reader.calls)
			}
		})
	}
}

func TestListDeploymentsByApplicationPropagatesMissingApplication(t *testing.T) {
	repository := &deploymentRepositoryFake{}
	reader := &applicationReaderFake{missing: true}
	_, err := NewListDeploymentsByApplication(repository, reader).Execute(
		context.Background(),
		1,
		10,
		0,
	)
	if !errors.Is(err, applications.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want applications.ErrNotFound", err)
	}
	if repository.listCalls != 0 {
		t.Fatalf("ListByApplication calls = %d, want 0", repository.listCalls)
	}
}
