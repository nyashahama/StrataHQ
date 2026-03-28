-- +goose Up

-- A billing period for a scheme (e.g. "October 2025").
CREATE TABLE levy_periods (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id   UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    label       TEXT        NOT NULL,
    amount_cents BIGINT     NOT NULL CHECK (amount_cents > 0),
    due_date    DATE        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_levy_periods_scheme_id ON levy_periods(scheme_id);

-- Per-unit levy obligation for one period.
CREATE TABLE levy_accounts (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id          UUID        NOT NULL REFERENCES units(id)        ON DELETE CASCADE,
    period_id        UUID        NOT NULL REFERENCES levy_periods(id) ON DELETE CASCADE,
    amount_cents     BIGINT      NOT NULL CHECK (amount_cents >= 0),
    paid_cents       BIGINT      NOT NULL DEFAULT 0 CHECK (paid_cents >= 0),
    status           TEXT        NOT NULL DEFAULT 'pending'
                                 CHECK (status IN ('paid', 'partial', 'overdue', 'pending')),
    due_date         DATE        NOT NULL,
    paid_date        DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (unit_id, period_id)
);

CREATE TRIGGER levy_accounts_set_updated_at
    BEFORE UPDATE ON levy_accounts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_levy_accounts_period_id ON levy_accounts(period_id);
CREATE INDEX idx_levy_accounts_unit_id   ON levy_accounts(unit_id);

-- Individual payments against a levy account.
-- reference is unique — idempotency key per architecture decision.
CREATE TABLE levy_payments (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    levy_account_id  UUID        NOT NULL REFERENCES levy_accounts(id) ON DELETE CASCADE,
    amount_cents     BIGINT      NOT NULL CHECK (amount_cents > 0),
    payment_date     DATE        NOT NULL,
    reference        TEXT        NOT NULL UNIQUE,
    bank_ref         TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_levy_payments_levy_account_id ON levy_payments(levy_account_id);

-- +goose Down

DROP INDEX  IF EXISTS idx_levy_payments_levy_account_id;
DROP TABLE  IF EXISTS levy_payments;
DROP TRIGGER IF EXISTS levy_accounts_set_updated_at ON levy_accounts;
DROP INDEX  IF EXISTS idx_levy_accounts_period_id;
DROP INDEX  IF EXISTS idx_levy_accounts_unit_id;
DROP TABLE  IF EXISTS levy_accounts;
DROP INDEX  IF EXISTS idx_levy_periods_scheme_id;
DROP TABLE  IF EXISTS levy_periods;
