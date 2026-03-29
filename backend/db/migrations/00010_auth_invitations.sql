-- backend/db/migrations/00010_auth_invitations.sql
-- +goose Up

-- Allow trustee/resident roles in org_memberships (previously only admin|agent).
ALTER TABLE org_memberships DROP CONSTRAINT IF EXISTS org_memberships_role_check;
ALTER TABLE org_memberships ADD CONSTRAINT org_memberships_role_check
    CHECK (role IN ('admin', 'agent', 'trustee', 'resident'));

-- Contact email for the managing agent org (set during onboarding wizard).
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS contact_email TEXT;

-- Invitations sent by agents to trustees and residents.
CREATE TABLE invitations (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        NOT NULL REFERENCES orgs(id)    ON DELETE CASCADE,
    scheme_id  UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    unit_id    UUID        REFERENCES units(id)            ON DELETE SET NULL,
    email      TEXT        NOT NULL,
    full_name  TEXT        NOT NULL,
    role       TEXT        NOT NULL CHECK (role IN ('trustee', 'resident')),
    token      TEXT        NOT NULL UNIQUE,
    status     TEXT        NOT NULL DEFAULT 'pending'
                           CHECK (status IN ('pending', 'accepted', 'revoked')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX invitations_org_status_idx ON invitations(org_id, status);

-- +goose Down

DROP TABLE IF EXISTS invitations;
ALTER TABLE orgs DROP COLUMN IF EXISTS contact_email;
ALTER TABLE org_memberships DROP CONSTRAINT IF EXISTS org_memberships_role_check;
ALTER TABLE org_memberships ADD CONSTRAINT org_memberships_role_check
    CHECK (role IN ('admin', 'agent'));
