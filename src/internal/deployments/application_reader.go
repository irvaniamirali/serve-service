package deployments

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

type applicationReader interface {
	GetApplicationByID(context.Context, int64) (applications.Application, error)
}
