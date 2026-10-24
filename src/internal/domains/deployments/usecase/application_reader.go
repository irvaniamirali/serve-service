package usecase

import (
	"context"

	"github.com/irvaniamirali/serve-service/internal/domains/applications"
)

type applicationReader interface {
	GetApplicationByID(context.Context, int64) (applications.Application, error)
}
