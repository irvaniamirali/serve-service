package ports

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/domains/applications"
)

// Repository defines persistence operations required by application use cases.
type Repository interface {
	CreateApplication(context.Context, applications.Application) (applications.Application, error)
	GetApplicationByID(context.Context, int64) (applications.Application, error)
	GetApplicationByGitHubRepositoryID(context.Context, int64) (applications.Application, error)
	ListApplications(context.Context, int, int) ([]applications.Application, error)
}
