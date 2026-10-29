package applications

import (
	"context"
	"errors"
	"testing"
)

func TestListApplicationsPassesPaginationToRepository(t *testing.T) {
	repository := &listApplicationsRepositoryFake{
		applications: []Application{{ID: 7, Name: "demo"}},
	}
	usecase := NewListApplications(repository)

	result, err := usecase.Execute(context.Background(), 20, 40)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result) != 1 || result[0].ID != 7 {
		t.Fatalf("Execute() = %+v, want repository results", result)
	}
	if repository.limit != 20 || repository.offset != 40 {
		t.Fatalf("repository pagination = %d/%d, want 20/40", repository.limit, repository.offset)
	}
}

func TestListApplicationsRejectsInvalidPagination(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{name: "zero limit", limit: 0},
		{name: "large limit", limit: MaxApplicationPageSize + 1},
		{name: "negative offset", limit: 1, offset: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &listApplicationsRepositoryFake{}
			_, err := NewListApplications(repository).Execute(context.Background(), test.limit, test.offset)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Execute() error = %v, want ErrValidation", err)
			}
			if repository.calls != 0 {
				t.Fatalf("ListApplications calls = %d, want 0", repository.calls)
			}
		})
	}
}

type listApplicationsRepositoryFake struct {
	applications []Application
	limit        int
	offset       int
	calls        int
}

func (repository *listApplicationsRepositoryFake) CreateApplication(
	context.Context,
	Application,
) (Application, error) {
	return Application{}, errors.New("not used")
}

func (repository *listApplicationsRepositoryFake) GetApplicationByID(
	context.Context,
	int64,
) (Application, error) {
	return Application{}, errors.New("not used")
}

func (repository *listApplicationsRepositoryFake) GetApplicationByGitHubRepositoryID(
	context.Context,
	int64,
) (Application, error) {
	return Application{}, errors.New("not used")
}

func (repository *listApplicationsRepositoryFake) ListApplications(
	_ context.Context,
	limit int,
	offset int,
) ([]Application, error) {
	repository.calls++
	repository.limit = limit
	repository.offset = offset
	return repository.applications, nil
}
