-- +goose Up

CREATE UNIQUE INDEX IF NOT EXISTS idx_early_access_requests_email_pending
ON early_access_requests (email)
WHERE status = 'pending';

-- +goose Down

DROP INDEX IF EXISTS idx_early_access_requests_email_pending;
