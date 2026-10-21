-- +goose Up
CREATE TABLE applications (
                              id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                              name TEXT NOT NULL UNIQUE,
                              repository_url TEXT NOT NULL,
                              created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE deployments (
                             id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                             application_id BIGINT NOT NULL
                                 REFERENCES applications (id) ON DELETE RESTRICT,
                             status TEXT NOT NULL DEFAULT 'queued'
                                 CHECK (
                                     status IN (
                                                'queued',
                                                'building',
                                                'deploying',
                                                'healthy',
                                                'failed',
                                                'cancelled'
                                         )
                                     ),
                             commit_sha TEXT,
                             error_message TEXT,
                             created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                             updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deployments_application_id_created_at
    ON deployments (application_id, created_at DESC);

-- +goose Down
DROP TABLE deployments;
DROP TABLE applications;
