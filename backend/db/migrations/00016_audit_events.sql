-- +goose Up

CREATE TABLE audit_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID        REFERENCES users(id) ON DELETE SET NULL,
    org_id        UUID        REFERENCES orgs(id) ON DELETE SET NULL,
    actor_role    TEXT,
    method        TEXT        NOT NULL,
    path          TEXT        NOT NULL,
    route_pattern TEXT        NOT NULL,
    status_code   INTEGER     NOT NULL CHECK (status_code >= 100 AND status_code <= 599),
    ip_address    TEXT        NOT NULL,
    user_agent    TEXT        NOT NULL DEFAULT '',
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_occurred_at ON audit_events (occurred_at DESC);
CREATE INDEX idx_audit_events_org_id_occurred_at ON audit_events (org_id, occurred_at DESC);
CREATE INDEX idx_audit_events_actor_user_id_occurred_at ON audit_events (actor_user_id, occurred_at DESC);
CREATE INDEX idx_audit_events_route_pattern_occurred_at ON audit_events (route_pattern, occurred_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_audit_events_route_pattern_occurred_at;
DROP INDEX IF EXISTS idx_audit_events_actor_user_id_occurred_at;
DROP INDEX IF EXISTS idx_audit_events_org_id_occurred_at;
DROP INDEX IF EXISTS idx_audit_events_occurred_at;
DROP TABLE IF EXISTS audit_events;
