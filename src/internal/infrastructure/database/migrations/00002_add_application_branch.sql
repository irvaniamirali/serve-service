-- +goose Up
ALTER TABLE applications
    ADD COLUMN branch TEXT NOT NULL DEFAULT 'main';

ALTER TABLE applications
    ALTER COLUMN branch DROP DEFAULT;

ALTER TABLE applications
    ADD CONSTRAINT applications_branch_not_empty CHECK (branch <> '');

ALTER TABLE deployments
    ADD CONSTRAINT deployments_commit_sha_format
        CHECK (commit_sha IS NULL OR commit_sha ~ '^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$');

DROP INDEX idx_deployments_application_id_created_at;
CREATE INDEX idx_deployments_application_history
    ON deployments (application_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX idx_deployments_application_history;
CREATE INDEX idx_deployments_application_id_created_at
    ON deployments (application_id, created_at DESC);

ALTER TABLE deployments
    DROP CONSTRAINT deployments_commit_sha_format;

ALTER TABLE applications
    DROP CONSTRAINT applications_branch_not_empty;

ALTER TABLE applications
    DROP COLUMN branch;
