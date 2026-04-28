-- name: CreateBankStatementImport :one
INSERT INTO bank_statement_imports (
    scheme_id,
    uploaded_by_user_id,
    bank_name,
    original_filename,
    raw_csv,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetBankStatementImport :one
SELECT *
FROM bank_statement_imports
WHERE id = $1 AND scheme_id = $2
LIMIT 1;

-- name: GetBankStatementImportByID :one
SELECT *
FROM bank_statement_imports
WHERE id = $1
LIMIT 1;

-- name: ListBankStatementImportsByScheme :many
SELECT *
FROM bank_statement_imports
WHERE scheme_id = $1
ORDER BY created_at DESC;

-- name: UpdateBankStatementImportStatus :one
UPDATE bank_statement_imports
SET status = $2,
    total_rows = $3,
    matched_rows = $4,
    ambiguous_rows = $5,
    unmatched_rows = $6,
    applied_rows = $7,
    parsed_at = $8,
    applied_at = $9,
    last_error = $10
WHERE id = $1
RETURNING *;

-- name: CreateBankStatementRow :one
INSERT INTO bank_statement_rows (
    import_id,
    row_number,
    transaction_date,
    amount_cents,
    reference,
    description,
    normalized_reference,
    row_fingerprint,
    status,
    confidence,
    match_reason,
    matched_levy_account_id,
    matched_levy_payment_id,
    raw_data
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: ListBankStatementRowsByImport :many
SELECT *
FROM bank_statement_rows
WHERE import_id = $1
ORDER BY row_number ASC;

-- name: UpdateBankStatementRowMatch :one
UPDATE bank_statement_rows
SET status = $2,
    confidence = $3,
    match_reason = $4,
    matched_levy_account_id = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateBankStatementRowApplied :one
UPDATE bank_statement_rows
SET status = 'applied',
    matched_levy_payment_id = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;
