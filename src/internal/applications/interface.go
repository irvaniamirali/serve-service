package applications

import (
	"context"
)

// Repository defines persistence operations required by application use cases.
type ApplicationRepositoryRepository interface {
	CreateApplication(context.Context, Application) (Application, error)
	GetApplicationByID(context.Context, int64) (Application, error)
	GetApplicationByGitHubRepositoryID(context.Context, int64) (Application, error)
	ListApplications(context.Context, int, int) ([]Application, error)
}
