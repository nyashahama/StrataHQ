-- backend/db/queries/invitations.sql

-- name: CreateInvitation :one
INSERT INTO invitations (org_id, scheme_id, unit_id, email, full_name, role, token, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetInvitationByToken :one
SELECT * FROM invitations WHERE token = $1 LIMIT 1;

-- name: GetInvitationByID :one
SELECT * FROM invitations WHERE id = $1 LIMIT 1;

-- name: ListInvitationsByOrg :many
SELECT * FROM invitations
WHERE org_id = $1 AND status = 'pending'
ORDER BY created_at DESC;

-- name: UpdateInvitationStatus :exec
UPDATE invitations SET status = $1 WHERE id = $2;

-- name: UpdateInvitationToken :one
UPDATE invitations
SET token = $1, expires_at = $2
WHERE id = $3
RETURNING *;
