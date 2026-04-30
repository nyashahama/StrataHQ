-- name: ListOpenAPISchemesByClient :many
SELECT s.*
FROM integration_api_client_schemes iacs
JOIN schemes s ON s.id = iacs.scheme_id
WHERE iacs.client_id = sqlc.arg(client_id)
ORDER BY s.name;

-- name: GetOpenAPISchemeByClient :one
SELECT s.*
FROM integration_api_client_schemes iacs
JOIN schemes s ON s.id = iacs.scheme_id
WHERE iacs.client_id = sqlc.arg(client_id)
  AND s.id = sqlc.arg(scheme_id)
LIMIT 1;

-- name: ListOpenAPIUnitsByScheme :many
SELECT *
FROM units
WHERE scheme_id = sqlc.arg(scheme_id)
ORDER BY identifier;

-- name: ListOpenAPILevyPeriodsByScheme :many
SELECT *
FROM levy_periods
WHERE scheme_id = sqlc.arg(scheme_id)
ORDER BY due_date DESC, created_at DESC;

-- name: ListOpenAPILevyAccountsByScheme :many
SELECT
    la.*,
    lp.scheme_id,
    lp.label AS period_label,
    u.identifier AS unit_identifier
FROM levy_accounts la
JOIN levy_periods lp ON lp.id = la.period_id
JOIN units u ON u.id = la.unit_id
WHERE lp.scheme_id = sqlc.arg(scheme_id)
  AND (sqlc.narg(period_id)::uuid IS NULL OR la.period_id = sqlc.narg(period_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR la.status = sqlc.narg(status)::text)
  AND (sqlc.narg(updated_since)::timestamptz IS NULL OR la.updated_at >= sqlc.narg(updated_since)::timestamptz)
ORDER BY lp.due_date DESC, u.identifier ASC
LIMIT sqlc.arg(limit_rows)
OFFSET sqlc.arg(offset_rows);

-- name: CountOpenAPILevyAccountsByScheme :one
SELECT COUNT(*)::int
FROM levy_accounts la
JOIN levy_periods lp ON lp.id = la.period_id
WHERE lp.scheme_id = sqlc.arg(scheme_id)
  AND (sqlc.narg(period_id)::uuid IS NULL OR la.period_id = sqlc.narg(period_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR la.status = sqlc.narg(status)::text)
  AND (sqlc.narg(updated_since)::timestamptz IS NULL OR la.updated_at >= sqlc.narg(updated_since)::timestamptz);

-- name: ListOpenAPILevyPaymentsByScheme :many
SELECT
    lpmt.*,
    lper.scheme_id,
    la.unit_id,
    u.identifier AS unit_identifier
FROM levy_payments lpmt
JOIN levy_accounts la ON la.id = lpmt.levy_account_id
JOIN levy_periods lper ON lper.id = la.period_id
JOIN units u ON u.id = la.unit_id
WHERE lper.scheme_id = sqlc.arg(scheme_id)
  AND (sqlc.narg(from_date)::date IS NULL OR lpmt.payment_date >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR lpmt.payment_date <= sqlc.narg(to_date)::date)
ORDER BY lpmt.payment_date DESC, lpmt.created_at DESC
LIMIT sqlc.arg(limit_rows)
OFFSET sqlc.arg(offset_rows);

-- name: CountOpenAPILevyPaymentsByScheme :one
SELECT COUNT(*)::int
FROM levy_payments lpmt
JOIN levy_accounts la ON la.id = lpmt.levy_account_id
JOIN levy_periods lper ON lper.id = la.period_id
WHERE lper.scheme_id = sqlc.arg(scheme_id)
  AND (sqlc.narg(from_date)::date IS NULL OR lpmt.payment_date >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR lpmt.payment_date <= sqlc.narg(to_date)::date);

-- name: ListOpenAPIBudgetLinesByScheme :many
SELECT *
FROM budget_lines
WHERE scheme_id = sqlc.arg(scheme_id)
  AND (sqlc.narg(period_label)::text IS NULL OR period_label = sqlc.narg(period_label)::text)
ORDER BY period_label DESC, category;

-- name: GetOpenAPIReserveFundByScheme :one
SELECT *
FROM reserve_fund
WHERE scheme_id = sqlc.arg(scheme_id)
LIMIT 1;
