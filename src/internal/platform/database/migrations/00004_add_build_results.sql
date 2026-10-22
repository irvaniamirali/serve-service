-- +goose Up
ALTER TABLE deployments
    ADD COLUMN image_reference TEXT,
    ADD COLUMN image_digest TEXT,
    ADD COLUMN build_started_at TIMESTAMPTZ,
    ADD COLUMN build_finished_at TIMESTAMPTZ,
    ADD COLUMN build_log TEXT;

ALTER TABLE deployments
    DROP CONSTRAINT IF EXISTS deployments_status_check;
ALTER TABLE deployments
    ADD CONSTRAINT deployments_status_check
    CHECK (status IN ('queued', 'building', 'built', 'deploying', 'healthy', 'failed', 'cancelled'));

ALTER TABLE deployments
    ADD CONSTRAINT deployments_build_log_size_check
    CHECK (build_log IS NULL OR length(build_log) <= 65536);

CREATE INDEX deployments_queued_builds_idx
    ON deployments (id)
    WHERE status = 'queued';

-- +goose Down
DROP INDEX IF EXISTS deployments_queued_builds_idx;
ALTER TABLE deployments
    DROP CONSTRAINT IF EXISTS deployments_build_log_size_check,
    DROP CONSTRAINT IF EXISTS deployments_status_check,
    DROP COLUMN IF EXISTS build_log,
    DROP COLUMN IF EXISTS build_finished_at,
    DROP COLUMN IF EXISTS build_started_at,
    DROP COLUMN IF EXISTS image_digest,
    DROP COLUMN IF EXISTS image_reference;

ALTER TABLE deployments
    ADD CONSTRAINT deployments_status_check
    CHECK (status IN ('queued', 'building', 'deploying', 'healthy', 'failed', 'cancelled'));
