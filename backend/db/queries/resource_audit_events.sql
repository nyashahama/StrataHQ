-- name: CreateResourceAuditEvent :one
INSERT INTO resource_audit_events (
    scheme_id,
    org_id,
    actor_user_id,
    actor_role,
    resource_type,
    resource_id,
    action,
    before_state,
    after_state,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: ListResourceAuditEventsByScheme :many
SELECT *
FROM resource_audit_events
WHERE scheme_id = $1
ORDER BY occurred_at DESC, id DESC
LIMIT $2;

-- name: ListResourceAuditEventsBySchemeAndAction :many
SELECT *
FROM resource_audit_events
WHERE scheme_id = $1
  AND action = $2
ORDER BY occurred_at DESC, id DESC
LIMIT $3;

-- name: ListResourceAuditEventsBySchemeAndOrg :many
SELECT *
FROM resource_audit_events
WHERE scheme_id = $1
  AND org_id = $2
ORDER BY occurred_at DESC, id DESC
LIMIT $3;

-- name: CountResourceAuditEventsBySchemeAndOrg :one
SELECT COUNT(*)
FROM resource_audit_events
WHERE scheme_id = $1
  AND org_id = $2;

-- name: CountResourceAuditEventsByScheme :one
SELECT COUNT(*)
FROM resource_audit_events
WHERE scheme_id = $1;
