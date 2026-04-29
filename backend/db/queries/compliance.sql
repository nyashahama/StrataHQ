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

-- name: GetComplianceItem :one
SELECT * FROM compliance_items WHERE id = $1 LIMIT 1;

-- name: UpdateComplianceItem :one
UPDATE compliance_items
SET status = $2, detail = $3, action = $4, due_date = $5, assessed_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteComplianceItem :exec
DELETE FROM compliance_items WHERE id = $1;

-- name: CountUpcomingDeadlinesByScheme :one
SELECT COUNT(*) FROM compliance_items
WHERE scheme_id = $1
  AND due_date IS NOT NULL
  AND due_date >= CURRENT_DATE
  AND due_date <= CURRENT_DATE + INTERVAL '30 days'
  AND status != 'compliant';

-- name: CreateComplianceAssessment :one
INSERT INTO compliance_assessments (scheme_id, score, total_items, compliant_count, at_risk_count, non_compliant_count)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListComplianceAssessmentsByScheme :many
SELECT * FROM compliance_assessments
WHERE scheme_id = $1
ORDER BY assessed_at DESC
LIMIT $2;
