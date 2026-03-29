-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: CreateOrg :one
INSERT INTO orgs (name)
VALUES ($1)
RETURNING *;

-- name: GetOrg :one
SELECT * FROM orgs
WHERE id = $1
LIMIT 1;

-- name: CreateOrgMembership :one
INSERT INTO org_memberships (user_id, org_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOrgMembershipByUser :one
SELECT * FROM org_memberships
WHERE user_id = $1 AND org_id = $2
LIMIT 1;

-- name: ListOrgMembershipsByUser :many
SELECT om.*, o.name AS org_name
FROM org_memberships om
JOIN orgs o ON o.id = om.org_id
WHERE om.user_id = $1
ORDER BY o.name;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1
  AND revoked = FALSE
  AND expires_at > NOW()
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE token = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE user_id = $1;

-- name: UpdateOrg :one
UPDATE orgs
SET name = $1, contact_email = $2
WHERE id = $3
RETURNING id, name;

-- name: ListSchemeMembershipsByUser :many
SELECT sm.scheme_id, s.name AS scheme_name, sm.unit_id, sm.role
FROM scheme_memberships sm
JOIN schemes s ON s.id = sm.scheme_id
WHERE sm.user_id = $1
ORDER BY s.name;
