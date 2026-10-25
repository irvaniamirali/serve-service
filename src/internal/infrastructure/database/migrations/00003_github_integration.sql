-- +goose Up
ALTER TABLE applications
    ADD COLUMN github_repository_id BIGINT,
    ADD COLUMN webhook_secret TEXT;

ALTER TABLE applications
    ADD CONSTRAINT applications_github_repository_id_not_negative
        CHECK (github_repository_id IS NULL OR github_repository_id > 0);

CREATE UNIQUE INDEX idx_applications_github_repository_id_unique
    ON applications (github_repository_id)
    WHERE github_repository_id IS NOT NULL;

CREATE TABLE github_webhook_deliveries (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    delivery_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    repo_id BIGINT,
    application_id BIGINT,
    outcome TEXT NOT NULL CHECK (outcome IN ('processing', 'processed', 'ignored', 'failed')),
    deployment_id BIGINT,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE github_webhook_deliveries;
DROP INDEX idx_applications_github_repository_id_unique;
ALTER TABLE applications
    DROP CONSTRAINT applications_github_repository_id_not_negative;
ALTER TABLE applications
    DROP COLUMN webhook_secret,
    DROP COLUMN github_repository_id;
