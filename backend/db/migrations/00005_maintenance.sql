-- +goose Up

CREATE TYPE maintenance_category AS ENUM (
    'plumbing', 'electrical', 'structural', 'garden', 'pool', 'other'
);

CREATE TYPE maintenance_status AS ENUM (
    'open', 'in_progress', 'pending_approval', 'resolved'
);

CREATE TABLE maintenance_requests (
    id                  UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id           UUID                 NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    unit_id             UUID                 REFERENCES units(id) ON DELETE SET NULL,
    title               TEXT                 NOT NULL,
    description         TEXT                 NOT NULL,
    category            maintenance_category NOT NULL,
    status              maintenance_status   NOT NULL DEFAULT 'open',
    contractor_name     TEXT,
    contractor_phone    TEXT,
    sla_hours           INTEGER              NOT NULL DEFAULT 48 CHECK (sla_hours > 0),
    submitted_by_unit   TEXT,
    created_at          TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    resolved_at         TIMESTAMPTZ
);

CREATE TRIGGER maintenance_requests_set_updated_at
    BEFORE UPDATE ON maintenance_requests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_maintenance_requests_scheme_id ON maintenance_requests(scheme_id);
CREATE INDEX idx_maintenance_requests_status    ON maintenance_requests(status);

-- +goose Down

DROP TRIGGER IF EXISTS maintenance_requests_set_updated_at ON maintenance_requests;
DROP INDEX   IF EXISTS idx_maintenance_requests_scheme_id;
DROP INDEX   IF EXISTS idx_maintenance_requests_status;
DROP TABLE   IF EXISTS maintenance_requests;
DROP TYPE    IF EXISTS maintenance_status;
DROP TYPE    IF EXISTS maintenance_category;
