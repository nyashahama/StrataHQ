-- +goose Up

CREATE TYPE early_access_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE early_access_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name   TEXT NOT NULL,
    email       TEXT NOT NULL,
    scheme_name TEXT NOT NULL,
    unit_count  INTEGER NOT NULL,
    status      early_access_status NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_early_access_requests_status ON early_access_requests (status);
CREATE INDEX idx_early_access_requests_email  ON early_access_requests (email);

-- +goose Down

DROP INDEX IF EXISTS idx_early_access_requests_email;
DROP INDEX IF EXISTS idx_early_access_requests_status;
DROP TABLE IF EXISTS early_access_requests;
DROP TYPE IF EXISTS early_access_status;
