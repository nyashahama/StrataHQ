-- name: UpsertBudgetLine :one
INSERT INTO budget_lines (scheme_id, category, period_label, budgeted_cents, actual_cents)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (scheme_id, category, period_label)
DO UPDATE SET
    budgeted_cents = EXCLUDED.budgeted_cents,
    actual_cents   = EXCLUDED.actual_cents
RETURNING *;

-- name: GetBudgetLine :one
SELECT * FROM budget_lines
WHERE scheme_id = $1 AND category = $2 AND period_label = $3
LIMIT 1;

-- name: ListBudgetLinesByScheme :many
SELECT * FROM budget_lines
WHERE scheme_id = $1
ORDER BY period_label DESC, category;

-- name: ListBudgetLinesByPeriod :many
SELECT * FROM budget_lines
WHERE scheme_id = $1 AND period_label = $2
ORDER BY category;

-- name: UpdateActualSpend :one
UPDATE budget_lines
SET actual_cents = $4
WHERE scheme_id = $1 AND category = $2 AND period_label = $3
RETURNING *;

-- name: UpsertReserveFund :one
INSERT INTO reserve_fund (scheme_id, balance_cents, target_cents, last_updated)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (scheme_id)
DO UPDATE SET
    balance_cents = EXCLUDED.balance_cents,
    target_cents  = EXCLUDED.target_cents,
    last_updated  = NOW()
RETURNING *;

-- name: GetReserveFund :one
SELECT * FROM reserve_fund
WHERE scheme_id = $1
LIMIT 1;
