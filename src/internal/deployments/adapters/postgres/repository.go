package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	applicationdomain "github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/deployments"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// deploymentModel maps deployment records to the existing database schema.
type deploymentModel struct {
	ID            int64     `gorm:"primaryKey;column:id"`
	ApplicationID int64     `gorm:"column:application_id"`
	Status        string    `gorm:"column:status;default:queued"`
	CommitSHA     *string   `gorm:"column:commit_sha"`
	ErrorMessage  *string   `gorm:"column:error_message"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (deploymentModel) TableName() string {
	return "deployments"
}

// Repository persists deployments using the shared GORM database handle.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (repository *Repository) Create(
	ctx context.Context,
	applicationID int64,
	commitSHA *string,
) (deployments.Deployment, error) {
	record := deploymentModel{
		ApplicationID: applicationID,
		CommitSHA:     commitSHA,
	}
	if err := repository.db.WithContext(ctx).Create(&record).Error; err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == "23503" &&
			postgresError.ConstraintName == "deployments_application_id_fkey" {
			return deployments.Deployment{}, applicationdomain.ErrNotFound
		}

		return deployments.Deployment{}, fmt.Errorf(
			"create deployment for application %d: %w",
			applicationID,
			err,
		)
	}

	return toDeployment(record), nil
}

func (repository *Repository) GetByID(
	ctx context.Context,
	id int64,
) (deployments.Deployment, error) {
	var record deploymentModel
	err := repository.db.WithContext(ctx).First(&record, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return deployments.Deployment{}, deployments.ErrNotFound
	}
	if err != nil {
		return deployments.Deployment{}, fmt.Errorf("get deployment %d: %w", id, err)
	}

	return toDeployment(record), nil
}

func (repository *Repository) ListByApplication(
	ctx context.Context,
	applicationID int64,
	limit int,
	offset int,
) ([]deployments.Deployment, error) {
	var records []deploymentModel
	err := repository.db.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf(
			"list deployments for application %d: %w",
			applicationID,
			err,
		)
	}

	result := make([]deployments.Deployment, 0, len(records))
	for _, record := range records {
		result = append(result, toDeployment(record))
	}

	return result, nil
}

func (repository *Repository) TransitionStatus(
	ctx context.Context,
	id int64,
	expectedStatus string,
	nextStatus string,
	failureMessage *string,
) (deployments.Deployment, error) {
	var record deploymentModel
	result := repository.db.WithContext(ctx).
		Model(&record).
		Clauses(clause.Returning{}).
		Where("id = ? AND status = ?", id, expectedStatus).
		Updates(map[string]any{
			"status":        nextStatus,
			"error_message": failureMessage,
			"updated_at":    gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return deployments.Deployment{}, fmt.Errorf("transition deployment %d status: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		var current deploymentModel
		err := repository.db.WithContext(ctx).First(&current, id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return deployments.Deployment{}, deployments.ErrNotFound
		}
		if err != nil {
			return deployments.Deployment{}, fmt.Errorf("check deployment %d status: %w", id, err)
		}

		return deployments.Deployment{}, deployments.ErrConcurrentTransition
	}

	return toDeployment(record), nil
}

func toDeployment(record deploymentModel) deployments.Deployment {
	return deployments.Deployment{
		ID:            record.ID,
		ApplicationID: record.ApplicationID,
		Status:        record.Status,
		CommitSHA:     record.CommitSHA,
		ErrorMessage:  record.ErrorMessage,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}
