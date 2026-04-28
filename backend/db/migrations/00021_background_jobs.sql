-- +goose Up
CREATE TYPE background_job_status AS ENUM ('queued', 'running', 'succeeded', 'failed');

CREATE TABLE background_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL,
    status background_job_status NOT NULL DEFAULT 'queued',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    idempotency_key TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    run_after TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    locked_by TEXT,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    succeeded_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    CONSTRAINT background_jobs_attempts_non_negative CHECK (attempts >= 0),
    CONSTRAINT background_jobs_max_attempts_positive CHECK (max_attempts > 0),
    CONSTRAINT background_jobs_idempotency_unique UNIQUE (kind, idempotency_key)
);

CREATE INDEX idx_background_jobs_claim
    ON background_jobs (run_after ASC, created_at ASC)
    WHERE status = 'queued';

CREATE INDEX idx_background_jobs_running_locked_at
    ON background_jobs (locked_at ASC)
    WHERE status = 'running';

CREATE INDEX idx_background_jobs_kind_status_created_at
    ON background_jobs (kind, status, created_at DESC);

ALTER TABLE collection_events
    DROP CONSTRAINT IF EXISTS collection_events_email_status_check,
    DROP CONSTRAINT IF EXISTS collection_events_whatsapp_status_check;

ALTER TABLE collection_events
    ADD CONSTRAINT collection_events_email_status_check
        CHECK (email_status IN ('queued', 'sent', 'failed', 'skipped') OR email_status IS NULL),
    ADD CONSTRAINT collection_events_whatsapp_status_check
        CHECK (whatsapp_status IN ('queued', 'sent', 'failed', 'skipped') OR whatsapp_status IS NULL);

-- +goose Down
ALTER TABLE collection_events
    DROP CONSTRAINT IF EXISTS collection_events_email_status_check,
    DROP CONSTRAINT IF EXISTS collection_events_whatsapp_status_check;

ALTER TABLE collection_events
    ADD CONSTRAINT collection_events_email_status_check
        CHECK (email_status IN ('sent', 'failed', 'skipped') OR email_status IS NULL),
    ADD CONSTRAINT collection_events_whatsapp_status_check
        CHECK (whatsapp_status IN ('sent', 'failed', 'skipped') OR whatsapp_status IS NULL);

DROP INDEX IF EXISTS idx_background_jobs_kind_status_created_at;
DROP INDEX IF EXISTS idx_background_jobs_running_locked_at;
DROP INDEX IF EXISTS idx_background_jobs_claim;
DROP TABLE IF EXISTS background_jobs;
DROP TYPE IF EXISTS background_job_status;
