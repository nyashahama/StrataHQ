-- +goose Up

CREATE TABLE schemes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    address    TEXT        NOT NULL,
    unit_count INTEGER     NOT NULL DEFAULT 0 CHECK (unit_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER schemes_set_updated_at
    BEFORE UPDATE ON schemes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_schemes_org_id ON schemes(org_id);

-- Physical units within a scheme.
-- section_value_bps: participation quota in basis points (417 = 4.17%).
CREATE TABLE units (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id           UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    identifier          TEXT        NOT NULL,
    owner_name          TEXT        NOT NULL,
    floor               INTEGER     NOT NULL DEFAULT 0,
    section_value_bps   INTEGER     NOT NULL CHECK (section_value_bps > 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scheme_id, identifier)
);

CREATE INDEX idx_units_scheme_id ON units(scheme_id);

-- Links authenticated users to a scheme with a role.
-- unit_id is NULL for trustees/agents who are not tied to a specific unit.
CREATE TABLE scheme_memberships (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    scheme_id  UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    unit_id    UUID        REFERENCES units(id)            ON DELETE SET NULL,
    role       TEXT        NOT NULL CHECK (role IN ('owner', 'trustee', 'resident')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, scheme_id)
);

CREATE INDEX idx_scheme_memberships_scheme_id ON scheme_memberships(scheme_id);

-- +goose Down

DROP TABLE IF EXISTS scheme_memberships;
DROP INDEX  IF EXISTS idx_units_scheme_id;
DROP TABLE  IF EXISTS units;
DROP TRIGGER IF EXISTS schemes_set_updated_at ON schemes;
DROP INDEX  IF EXISTS idx_schemes_org_id;
DROP TABLE  IF EXISTS schemes;
