package applications

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Repository defines the persistence operations for applications.
type Repository interface {
	CreateApplication(context.Context, Application) (Application, error)
	GetApplicationByID(context.Context, int64) (Application, error)
}

// ApplicationRepository persists applications using GORM and PostgreSQL.
type ApplicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// CreateApplication persists a new application.
func (repository *ApplicationRepository) CreateApplication(
	ctx context.Context,
	application Application,
) (Application, error) {
	record := ApplicationModel{
		Name:          application.Name,
		RepositoryURL: application.RepositoryURL,
		Branch:        application.Branch,
	}

	err := repository.db.WithContext(ctx).Create(&record).Error
	if err != nil {
		var postgresError *pgconn.PgError

		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" &&
			postgresError.ConstraintName == "applications_name_key" {
			return Application{}, ErrConflict
		}

		return Application{}, fmt.Errorf("create application: %w", err)
	}

	return toApplication(record), nil
}

// GetApplicationByID retrieves an application by its identifier.
func (repository *ApplicationRepository) GetApplicationByID(
	ctx context.Context,
	id int64,
) (Application, error) {
	var record ApplicationModel

	err := repository.db.WithContext(ctx).First(&record, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Application{}, ErrNotFound
	}

	if err != nil {
		return Application{}, fmt.Errorf(
			"get application %d: %w",
			id,
			err,
		)
	}

	return toApplication(record), nil
}

// toApplication converts a database record into the domain model.
func toApplication(record ApplicationModel) Application {
	return Application{
		ID:            record.ID,
		Name:          record.Name,
		RepositoryURL: record.RepositoryURL,
		Branch:        record.Branch,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}
