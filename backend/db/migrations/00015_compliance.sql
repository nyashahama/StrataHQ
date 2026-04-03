-- +goose Up

CREATE TYPE compliance_status AS ENUM ('compliant', 'at-risk', 'non-compliant');
CREATE TYPE compliance_category AS ENUM ('financial', 'governance', 'administrative', 'insurance');

CREATE TABLE compliance_items (
    id           UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id    UUID                NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    category     compliance_category NOT NULL,
    title        TEXT                NOT NULL,
    requirement  TEXT                NOT NULL,
    status       compliance_status   NOT NULL DEFAULT 'compliant',
    detail       TEXT                NOT NULL,
    action       TEXT                NOT NULL,
    due_date     DATE,
    assessed_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

CREATE TRIGGER compliance_items_set_updated_at
    BEFORE UPDATE ON compliance_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_compliance_items_scheme_id ON compliance_items(scheme_id);
CREATE INDEX idx_compliance_items_category ON compliance_items(scheme_id, category);
CREATE INDEX idx_compliance_items_status ON compliance_items(scheme_id, status);

-- +goose Down

DROP INDEX IF EXISTS idx_compliance_items_status;
DROP INDEX IF EXISTS idx_compliance_items_category;
DROP INDEX IF EXISTS idx_compliance_items_scheme_id;
DROP TRIGGER IF EXISTS compliance_items_set_updated_at ON compliance_items;
DROP TABLE IF EXISTS compliance_items;
DROP TYPE IF EXISTS compliance_category;
DROP TYPE IF EXISTS compliance_status;
