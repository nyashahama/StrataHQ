-- name: CreateLevyPeriod :one
INSERT INTO levy_periods (scheme_id, label, amount_cents, due_date)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLevyPeriod :one
SELECT * FROM levy_periods
WHERE id = $1
LIMIT 1;

-- name: ListLevyPeriodsByScheme :many
SELECT * FROM levy_periods
WHERE scheme_id = $1
ORDER BY due_date DESC;

-- name: CreateLevyAccount :one
INSERT INTO levy_accounts (unit_id, period_id, amount_cents, due_date)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLevyAccount :one
SELECT * FROM levy_accounts
WHERE id = $1
LIMIT 1;

-- name: GetLevyAccountByUnitAndPeriod :one
SELECT * FROM levy_accounts
WHERE unit_id = $1 AND period_id = $2
LIMIT 1;

-- name: ListLevyAccountsByPeriod :many
SELECT la.*, u.identifier AS unit_identifier, u.owner_name
FROM levy_accounts la
JOIN units u ON u.id = la.unit_id
WHERE la.period_id = $1
ORDER BY u.identifier;

-- name: UpdateLevyAccountPaid :one
UPDATE levy_accounts
SET paid_cents = $2,
    status     = $3,
    paid_date  = $4
WHERE id = $1
RETURNING *;

-- name: CreateLevyPayment :one
INSERT INTO levy_payments (levy_account_id, amount_cents, payment_date, reference, bank_ref)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLevyPaymentByReference :one
SELECT * FROM levy_payments
WHERE reference = $1
LIMIT 1;

-- name: ListLevyPaymentsByAccount :many
SELECT * FROM levy_payments
WHERE levy_account_id = $1
ORDER BY payment_date DESC;
