package deployments

import (
	"context"
	"errors"
	"fmt"

	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines persistence operations for deployments.
type Repository interface {
	Create(context.Context, int64, *string) (Deployment, error)
	GetByID(context.Context, int64) (Deployment, error)
	ListByApplication(context.Context, int64, int, int) ([]Deployment, error)
	TransitionStatus(context.Context, int64, string, string, *string) (Deployment, error)
}

// PostgresRepository persists deployments using the shared GORM database handle.
type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (repository *PostgresRepository) Create(
	ctx context.Context,
	applicationID int64,
	commitSHA *string,
) (Deployment, error) {
	record := DeploymentModel{
		ApplicationID: applicationID,
		CommitSHA:     commitSHA,
	}
	if err := repository.db.WithContext(ctx).Create(&record).Error; err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == "23503" &&
			postgresError.ConstraintName == "deployments_application_id_fkey" {
			return Deployment{}, applications.ErrNotFound
		}

		return Deployment{}, fmt.Errorf(
			"create deployment for application %d: %w",
			applicationID,
			err,
		)
	}

	return toDeployment(record), nil
}

func (repository *PostgresRepository) GetByID(
	ctx context.Context,
	id int64,
) (Deployment, error) {
	var record DeploymentModel
	err := repository.db.WithContext(ctx).First(&record, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Deployment{}, ErrNotFound
	}
	if err != nil {
		return Deployment{}, fmt.Errorf("get deployment %d: %w", id, err)
	}

	return toDeployment(record), nil
}

func (repository *PostgresRepository) ListByApplication(
	ctx context.Context,
	applicationID int64,
	limit int,
	offset int,
) ([]Deployment, error) {
	var records []DeploymentModel
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

	result := make([]Deployment, 0, len(records))
	for _, record := range records {
		result = append(result, toDeployment(record))
	}

	return result, nil
}

func (repository *PostgresRepository) TransitionStatus(
	ctx context.Context,
	id int64,
	expectedStatus string,
	nextStatus string,
	failureMessage *string,
) (Deployment, error) {
	var record DeploymentModel
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
		return Deployment{}, fmt.Errorf("transition deployment %d status: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		var current DeploymentModel
		err := repository.db.WithContext(ctx).First(&current, id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Deployment{}, ErrNotFound
		}
		if err != nil {
			return Deployment{}, fmt.Errorf("check deployment %d status: %w", id, err)
		}

		return Deployment{}, ErrConcurrentTransition
	}

	return toDeployment(record), nil
}

func toDeployment(record DeploymentModel) Deployment {
	return Deployment{
		ID:            record.ID,
		ApplicationID: record.ApplicationID,
		Status:        record.Status,
		CommitSHA:     record.CommitSHA,
		ErrorMessage:  record.ErrorMessage,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}
