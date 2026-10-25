package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/irvaniamirali/serve-service/internal/domains/applications"
	"github.com/irvaniamirali/serve-service/internal/domains/deployments"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// deploymentModel maps deployment records to the existing database schema.
type deploymentModel struct {
	ID              int64      `gorm:"primaryKey;column:id"`
	ApplicationID   int64      `gorm:"column:application_id"`
	Status          string     `gorm:"column:status;default:queued"`
	CommitSHA       *string    `gorm:"column:commit_sha"`
	ErrorMessage    *string    `gorm:"column:error_message"`
	ImageReference  *string    `gorm:"column:image_reference"`
	ImageDigest     *string    `gorm:"column:image_digest"`
	BuildStartedAt  *time.Time `gorm:"column:build_started_at"`
	BuildFinishedAt *time.Time `gorm:"column:build_finished_at"`
	BuildLog        *string    `gorm:"column:build_log"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (deploymentModel) TableName() string {
	return "deployments"
}

// Repository persists deployments using the shared GORM database handle.
type Repository struct {
	db *gorm.DB
}

const (
	buildQueueAdvisoryLock int64 = 0x535256434255494c
	staleBuildThreshold          = 30 * time.Minute
)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ClaimNextQueued atomically moves one queued deployment into building state.
func (repository *Repository) ClaimNextQueued(ctx context.Context) (deployments.Deployment, error) {
	var claimed deployments.Deployment
	noQueuedBuild := false
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", buildQueueAdvisoryLock).Error; err != nil {
			return fmt.Errorf("lock deployment build queue: %w", err)
		}

		now := time.Now().UTC()
		staleBefore := now.Add(-staleBuildThreshold)
		if err := tx.Model(&deploymentModel{}).
			Where(
				"status = ? AND (build_started_at IS NULL OR build_started_at < ?)",
				deployments.StatusBuilding,
				staleBefore,
			).
			Updates(map[string]any{
				"status":            deployments.StatusFailed,
				"error_message":     "build interrupted; create a new deployment to retry",
				"build_finished_at": now,
				"updated_at":        now,
			}).Error; err != nil {
			return fmt.Errorf("fail stale deployment builds: %w", err)
		}
		var activeBuilds int64
		if err := tx.Model(&deploymentModel{}).
			Where("status = ?", deployments.StatusBuilding).
			Count(&activeBuilds).Error; err != nil {
			return fmt.Errorf("count active deployment builds: %w", err)
		}
		if activeBuilds > 0 {
			noQueuedBuild = true
			return nil
		}

		var record deploymentModel
		findResult := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", deployments.StatusQueued).
			Order("id ASC").
			Limit(1).
			Find(&record)
		if findResult.Error != nil {
			return fmt.Errorf("find queued deployment: %w", findResult.Error)
		}
		if findResult.RowsAffected == 0 {
			noQueuedBuild = true
			return nil
		}

		startedAt := now
		result := tx.Model(&deploymentModel{}).
			Where("id = ? AND status = ?", record.ID, deployments.StatusQueued).
			Updates(map[string]any{
				"status":            deployments.StatusBuilding,
				"build_started_at":  startedAt,
				"build_finished_at": nil,
				"build_log":         nil,
				"image_reference":   nil,
				"image_digest":      nil,
				"error_message":     nil,
				"updated_at":        startedAt,
			})
		if result.Error != nil {
			return fmt.Errorf("claim queued deployment %d: %w", record.ID, result.Error)
		}
		if result.RowsAffected != 1 {
			return deployments.ErrConcurrentTransition
		}

		record.Status = deployments.StatusBuilding
		record.BuildStartedAt = &startedAt
		record.BuildFinishedAt = nil
		record.BuildLog = nil
		record.ImageReference = nil
		record.ImageDigest = nil
		record.ErrorMessage = nil
		record.UpdatedAt = startedAt
		claimed = toDeployment(record)
		return nil
	})
	if err != nil {
		return deployments.Deployment{}, err
	}
	if noQueuedBuild {
		return deployments.Deployment{}, deployments.ErrNoQueuedBuild
	}

	return claimed, nil
}

// RecordBuildResult persists a terminal build result only while the deployment is building.
func (repository *Repository) RecordBuildResult(
	ctx context.Context,
	id int64,
	succeeded bool,
	imageReference *string,
	imageDigest *string,
	buildLog string,
	failureMessage *string,
) (deployments.Deployment, error) {
	status := deployments.StatusFailed
	if succeeded {
		status = deployments.StatusBuilt
		failureMessage = nil
	} else {
		imageReference = nil
		imageDigest = nil
	}
	buildLog = truncateBuildLog(strings.ToValidUTF8(buildLog, "\uFFFD"))
	finishedAt := time.Now().UTC()

	var record deploymentModel
	result := repository.db.WithContext(ctx).
		Model(&record).
		Clauses(clause.Returning{}).
		Where("id = ? AND status = ?", id, deployments.StatusBuilding).
		Updates(map[string]any{
			"status":            status,
			"image_reference":   imageReference,
			"image_digest":      imageDigest,
			"build_finished_at": finishedAt,
			"build_log":         buildLog,
			"error_message":     failureMessage,
			"updated_at":        finishedAt,
		})
	if result.Error != nil {
		return deployments.Deployment{}, fmt.Errorf("record build result for deployment %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		var current deploymentModel
		err := repository.db.WithContext(ctx).First(&current, id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return deployments.Deployment{}, deployments.ErrNotFound
		}
		if err != nil {
			return deployments.Deployment{}, fmt.Errorf("check deployment %d build status: %w", id, err)
		}

		return deployments.Deployment{}, deployments.ErrConcurrentTransition
	}

	return toDeployment(record), nil
}

const maxPersistedBuildLogBytes = 64 * 1024

func truncateBuildLog(value string) string {
	if len(value) <= maxPersistedBuildLogBytes {
		return value
	}

	return strings.ToValidUTF8(value[len(value)-maxPersistedBuildLogBytes:], "\uFFFD")
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
			return deployments.Deployment{}, applications.ErrNotFound
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
		ID:              record.ID,
		ApplicationID:   record.ApplicationID,
		Status:          record.Status,
		CommitSHA:       record.CommitSHA,
		ErrorMessage:    record.ErrorMessage,
		ImageReference:  record.ImageReference,
		ImageDigest:     record.ImageDigest,
		BuildStartedAt:  record.BuildStartedAt,
		BuildFinishedAt: record.BuildFinishedAt,
		BuildLog:        record.BuildLog,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}
