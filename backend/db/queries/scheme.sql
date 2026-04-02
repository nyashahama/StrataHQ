-- name: CreateScheme :one
INSERT INTO schemes (org_id, name, address, unit_count)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetScheme :one
SELECT * FROM schemes
WHERE id = $1
LIMIT 1;

-- name: ListSchemesByOrg :many
SELECT * FROM schemes
WHERE org_id = $1
ORDER BY name;

-- name: UpdateScheme :one
UPDATE schemes
SET name = $2, address = $3, unit_count = $4
WHERE id = $1
RETURNING *;

-- name: DeleteScheme :exec
DELETE FROM schemes
WHERE id = $1;

-- name: CreateUnit :one
INSERT INTO units (scheme_id, identifier, owner_name, floor, section_value_bps)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUnit :one
SELECT * FROM units
WHERE id = $1
LIMIT 1;

-- name: ListUnitsByScheme :many
SELECT * FROM units
WHERE scheme_id = $1
ORDER BY identifier;

-- name: UpdateUnit :one
UPDATE units
SET identifier = $2,
    owner_name = $3,
    floor = $4,
    section_value_bps = $5
WHERE id = $1
RETURNING *;

-- name: UpsertSchemeMembership :one
INSERT INTO scheme_memberships (user_id, scheme_id, unit_id, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, scheme_id)
DO UPDATE SET unit_id = EXCLUDED.unit_id, role = EXCLUDED.role
RETURNING *;

-- name: GetSchemeMembership :one
SELECT * FROM scheme_memberships
WHERE user_id = $1 AND scheme_id = $2
LIMIT 1;

-- name: ListSchemeMembersByScheme :many
SELECT sm.*, u.full_name, u.email
FROM scheme_memberships sm
JOIN users u ON u.id = sm.user_id
WHERE sm.scheme_id = $1
ORDER BY u.full_name;

-- name: DeleteSchemeMembership :exec
DELETE FROM scheme_memberships
WHERE user_id = $1 AND scheme_id = $2;
