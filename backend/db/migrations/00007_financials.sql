-- +goose Up

-- Annual budget lines per scheme — one row per (scheme, category, period).
CREATE TABLE budget_lines (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id       UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    category        TEXT        NOT NULL,
    period_label    TEXT        NOT NULL,
    budgeted_cents  BIGINT      NOT NULL DEFAULT 0 CHECK (budgeted_cents >= 0),
    actual_cents    BIGINT      NOT NULL DEFAULT 0 CHECK (actual_cents >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scheme_id, category, period_label)
);

CREATE TRIGGER budget_lines_set_updated_at
    BEFORE UPDATE ON budget_lines
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_budget_lines_scheme_id ON budget_lines(scheme_id);

-- One reserve fund row per scheme; upserted as balance changes.
CREATE TABLE reserve_fund (
    scheme_id    UUID        PRIMARY KEY REFERENCES schemes(id) ON DELETE CASCADE,
    balance_cents BIGINT     NOT NULL DEFAULT 0 CHECK (balance_cents >= 0),
    target_cents  BIGINT     NOT NULL DEFAULT 0 CHECK (target_cents >= 0),
    last_updated  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down

DROP TRIGGER IF EXISTS budget_lines_set_updated_at ON budget_lines;
DROP INDEX   IF EXISTS idx_budget_lines_scheme_id;
DROP TABLE   IF EXISTS budget_lines;
DROP TABLE   IF EXISTS reserve_fund;
