package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// applicationModel maps application records to the existing database schema.
type applicationModel struct {
	ID                 int64     `gorm:"primaryKey;column:id"`
	Name               string    `gorm:"column:name"`
	RepositoryURL      string    `gorm:"column:repository_url"`
	Branch             string    `gorm:"column:branch"`
	GitHubRepositoryID *int64    `gorm:"column:github_repository_id"`
	WebhookSecret      string    `gorm:"column:webhook_secret"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (applicationModel) TableName() string {
	return "applications"
}

// Repository persists applications using the shared GORM database handle.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateApplication persists a new application.
func (repository *Repository) CreateApplication(
	ctx context.Context,
	application applications.Application,
) (applications.Application, error) {
	record := applicationModel{
		Name:               application.Name,
		RepositoryURL:      application.RepositoryURL,
		Branch:             application.Branch,
		GitHubRepositoryID: application.GitHubRepositoryID,
		WebhookSecret:      application.WebhookSecret,
	}

	err := repository.db.WithContext(ctx).Create(&record).Error
	if err != nil {
		var postgresError *pgconn.PgError

		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" &&
			(postgresError.ConstraintName == "applications_name_key" ||
				postgresError.ConstraintName == "applications_github_repository_id_key") {
			return applications.Application{}, applications.ErrConflict
		}

		return applications.Application{}, fmt.Errorf("create application: %w", err)
	}

	return toApplication(record), nil
}

// GetApplicationByID retrieves an application by its identifier.
func (repository *Repository) GetApplicationByID(
	ctx context.Context,
	id int64,
) (applications.Application, error) {
	var record applicationModel

	err := repository.db.WithContext(ctx).First(&record, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return applications.Application{}, applications.ErrNotFound
	}

	if err != nil {
		return applications.Application{}, fmt.Errorf(
			"get application %d: %w",
			id,
			err,
		)
	}

	return toApplication(record), nil
}

func (repository *Repository) GetApplicationByGitHubRepositoryID(
	ctx context.Context,
	githubRepositoryID int64,
) (applications.Application, error) {
	var record applicationModel

	err := repository.db.WithContext(ctx).Where("github_repository_id = ?", githubRepositoryID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return applications.Application{}, applications.ErrNotFound
	}
	if err != nil {
		return applications.Application{}, fmt.Errorf(
			"get application by GitHub repository ID %d: %w",
			githubRepositoryID,
			err,
		)
	}

	return toApplication(record), nil
}

// ListApplications returns a stable newest-first page of registered applications.
func (repository *Repository) ListApplications(
	ctx context.Context,
	limit int,
	offset int,
) ([]applications.Application, error) {
	var records []applicationModel
	if err := repository.db.WithContext(ctx).
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}

	result := make([]applications.Application, 0, len(records))
	for _, record := range records {
		result = append(result, toApplication(record))
	}

	return result, nil
}

// toApplication converts a database record into the domain model.
func toApplication(record applicationModel) applications.Application {
	return applications.Application{
		ID:                 record.ID,
		Name:               record.Name,
		RepositoryURL:      record.RepositoryURL,
		Branch:             record.Branch,
		GitHubRepositoryID: record.GitHubRepositoryID,
		WebhookSecret:      record.WebhookSecret,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
}
