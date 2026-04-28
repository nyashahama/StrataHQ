-- +goose Up
CREATE TYPE bank_statement_import_status AS ENUM (
    'queued',
    'processing',
    'review_required',
    'applied',
    'failed'
);

CREATE TYPE bank_statement_row_status AS ENUM (
    'matched',
    'ambiguous',
    'unmatched',
    'applied',
    'skipped',
    'failed'
);

CREATE TABLE bank_statement_imports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id UUID NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    uploaded_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    bank_name TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    raw_csv BYTEA NOT NULL,
    status bank_statement_import_status NOT NULL DEFAULT 'queued',
    total_rows INTEGER NOT NULL DEFAULT 0,
    matched_rows INTEGER NOT NULL DEFAULT 0,
    ambiguous_rows INTEGER NOT NULL DEFAULT 0,
    unmatched_rows INTEGER NOT NULL DEFAULT 0,
    applied_rows INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    parsed_at TIMESTAMPTZ,
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT bank_statement_imports_total_rows_non_negative CHECK (total_rows >= 0),
    CONSTRAINT bank_statement_imports_matched_rows_non_negative CHECK (matched_rows >= 0),
    CONSTRAINT bank_statement_imports_ambiguous_rows_non_negative CHECK (ambiguous_rows >= 0),
    CONSTRAINT bank_statement_imports_unmatched_rows_non_negative CHECK (unmatched_rows >= 0),
    CONSTRAINT bank_statement_imports_applied_rows_non_negative CHECK (applied_rows >= 0)
);

CREATE TABLE bank_statement_rows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    import_id UUID NOT NULL REFERENCES bank_statement_imports(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL,
    transaction_date DATE NOT NULL,
    amount_cents BIGINT NOT NULL,
    reference TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    normalized_reference TEXT NOT NULL DEFAULT '',
    row_fingerprint TEXT NOT NULL,
    status bank_statement_row_status NOT NULL DEFAULT 'unmatched',
    confidence INTEGER NOT NULL DEFAULT 0,
    match_reason TEXT,
    matched_levy_account_id UUID REFERENCES levy_accounts(id) ON DELETE SET NULL,
    matched_levy_payment_id UUID REFERENCES levy_payments(id) ON DELETE SET NULL,
    raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT bank_statement_rows_confidence_range CHECK (confidence >= 0 AND confidence <= 100),
    CONSTRAINT bank_statement_rows_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT bank_statement_rows_unique_row UNIQUE (import_id, row_number),
    CONSTRAINT bank_statement_rows_unique_fingerprint UNIQUE (row_fingerprint)
);

CREATE INDEX idx_bank_statement_imports_scheme_created_at
    ON bank_statement_imports (scheme_id, created_at DESC);

CREATE INDEX idx_bank_statement_imports_status
    ON bank_statement_imports (scheme_id, status, created_at DESC);

CREATE INDEX idx_bank_statement_rows_import_status
    ON bank_statement_rows (import_id, status, row_number);

CREATE INDEX idx_bank_statement_rows_matched_account
    ON bank_statement_rows (matched_levy_account_id)
    WHERE matched_levy_account_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_bank_statement_rows_matched_account;
DROP INDEX IF EXISTS idx_bank_statement_rows_import_status;
DROP INDEX IF EXISTS idx_bank_statement_imports_status;
DROP INDEX IF EXISTS idx_bank_statement_imports_scheme_created_at;
DROP TABLE IF EXISTS bank_statement_rows;
DROP TABLE IF EXISTS bank_statement_imports;
DROP TYPE IF EXISTS bank_statement_row_status;
DROP TYPE IF EXISTS bank_statement_import_status;
