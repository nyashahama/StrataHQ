-- +goose Up
ALTER TABLE background_jobs DROP CONSTRAINT IF EXISTS background_jobs_idempotency_unique;
CREATE UNIQUE INDEX background_jobs_idempotency_unique ON background_jobs (kind, idempotency_key) WHERE status IN ('queued', 'running');

-- +goose Down
DROP INDEX IF EXISTS background_jobs_idempotency_unique;
ALTER TABLE background_jobs ADD CONSTRAINT background_jobs_idempotency_unique UNIQUE (kind, idempotency_key);
