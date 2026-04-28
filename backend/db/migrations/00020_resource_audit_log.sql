-- +goose Up

CREATE TABLE resource_audit_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id     UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    org_id        UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    actor_user_id UUID        REFERENCES users(id) ON DELETE SET NULL,
    actor_role    TEXT        NOT NULL DEFAULT '',
    resource_type TEXT        NOT NULL,
    resource_id   UUID,
    action        TEXT        NOT NULL,
    before_state  JSONB       NOT NULL DEFAULT '{}'::jsonb,
    after_state   JSONB       NOT NULL DEFAULT '{}'::jsonb,
    metadata      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (resource_type <> ''),
    CHECK (action <> '')
);

CREATE INDEX idx_resource_audit_events_scheme_occurred_at
    ON resource_audit_events (scheme_id, occurred_at DESC);
CREATE INDEX idx_resource_audit_events_org_occurred_at
    ON resource_audit_events (org_id, occurred_at DESC);
CREATE INDEX idx_resource_audit_events_actor_occurred_at
    ON resource_audit_events (actor_user_id, occurred_at DESC)
    WHERE actor_user_id IS NOT NULL;
CREATE INDEX idx_resource_audit_events_resource
    ON resource_audit_events (resource_type, resource_id, occurred_at DESC)
    WHERE resource_id IS NOT NULL;
CREATE INDEX idx_resource_audit_events_action_occurred_at
    ON resource_audit_events (action, occurred_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_resource_audit_events_action_occurred_at;
DROP INDEX IF EXISTS idx_resource_audit_events_resource;
DROP INDEX IF EXISTS idx_resource_audit_events_actor_occurred_at;
DROP INDEX IF EXISTS idx_resource_audit_events_org_occurred_at;
DROP INDEX IF EXISTS idx_resource_audit_events_scheme_occurred_at;
DROP TABLE IF EXISTS resource_audit_events;
