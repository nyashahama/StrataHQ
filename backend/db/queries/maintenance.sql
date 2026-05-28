-- name: CreateMaintenanceRequest :one
INSERT INTO maintenance_requests (
    scheme_id, unit_id, title, description, category, sla_hours, submitted_by_unit
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetMaintenanceRequest :one
SELECT * FROM maintenance_requests
WHERE id = $1
LIMIT 1;

-- name: ListMaintenanceRequestsByScheme :many
SELECT * FROM maintenance_requests
WHERE scheme_id = $1
ORDER BY created_at DESC;

-- name: ListMaintenanceRequestsDetailedByScheme :many
SELECT mr.*, u.identifier AS unit_identifier, u.owner_name
FROM maintenance_requests mr
LEFT JOIN units u ON u.id = mr.unit_id
WHERE mr.scheme_id = $1
ORDER BY mr.created_at DESC;

-- name: ListOpenMaintenanceRequestsByScheme :many
SELECT * FROM maintenance_requests
WHERE scheme_id = $1
  AND status != 'resolved'
ORDER BY created_at DESC;

-- name: UpdateMaintenanceStatus :one
UPDATE maintenance_requests
SET status = $2
WHERE id = $1
RETURNING *;

-- name: AssignMaintenanceContractor :one
UPDATE maintenance_requests
SET contractor_id    = NULL,
    contractor_name  = $2,
    contractor_phone = $3,
    status           = 'in_progress',
    resolved_at      = NULL
WHERE id = $1
RETURNING *;

-- name: ResolveMaintenanceRequest :one
UPDATE maintenance_requests
SET status      = 'resolved',
    resolved_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountOpenMaintenanceByScheme :one
SELECT COUNT(*) FROM maintenance_requests
WHERE scheme_id = $1
  AND status != 'resolved';

-- name: CountSlaBreachedMaintenanceByScheme :one
SELECT COUNT(*) FROM maintenance_requests
WHERE scheme_id = $1
  AND status NOT IN ('resolved', 'pending_approval')
  AND created_at + (sla_hours || ' hours')::interval < NOW();

-- name: AssignMaintenanceContractorProfile :one
UPDATE maintenance_requests
SET contractor_id    = sqlc.arg(contractor_id),
    contractor_name  = sqlc.arg(contractor_name),
    contractor_phone = sqlc.arg(contractor_phone),
    status           = 'in_progress',
    resolved_at      = NULL
WHERE id = sqlc.arg(id)
RETURNING *;
