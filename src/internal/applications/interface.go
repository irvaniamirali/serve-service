package applications

import (
	"context"
)

// ApplicationRepositoryInterface defines persistence operations required by application use cases.
type ApplicationRepositoryInterface interface {
	CreateApplication(context.Context, Application) (Application, error)
	GetApplicationByID(context.Context, int64) (Application, error)
	GetApplicationByGitHubRepositoryID(context.Context, int64) (Application, error)
	ListApplications(context.Context, int, int) ([]Application, error)
}
