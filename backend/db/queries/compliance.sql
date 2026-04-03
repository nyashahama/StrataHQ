-- name: CreateComplianceItem :one
INSERT INTO compliance_items (
    scheme_id, category, title, requirement, status, detail, action, due_date, assessed_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListComplianceItemsByScheme :many
SELECT *
FROM compliance_items
WHERE scheme_id = $1
ORDER BY category ASC, title ASC;
