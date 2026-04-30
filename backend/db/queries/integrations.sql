-- name: CreateIntegrationAPIClient :one
INSERT INTO integration_api_clients (
    org_id,
    name,
    key_prefix,
    key_hash,
    scopes,
    created_by_user_id,
    expires_at
)
VALUES (
    sqlc.arg(org_id),
    sqlc.arg(name),
    sqlc.arg(key_prefix),
    sqlc.arg(key_hash),
    sqlc.arg(scopes),
    sqlc.arg(created_by_user_id),
    sqlc.arg(expires_at)
)
RETURNING *;

-- name: LinkIntegrationAPIClientScheme :exec
INSERT INTO integration_api_client_schemes (client_id, scheme_id)
VALUES (sqlc.arg(client_id), sqlc.arg(scheme_id))
ON CONFLICT DO NOTHING;

-- name: GetIntegrationAPIClientByPrefix :one
SELECT *
FROM integration_api_clients
WHERE key_prefix = sqlc.arg(key_prefix)
LIMIT 1;

-- name: ListIntegrationAPIClientSchemes :many
SELECT scheme_id
FROM integration_api_client_schemes
WHERE client_id = sqlc.arg(client_id)
ORDER BY scheme_id;

-- name: ListIntegrationAPIClientsByOrg :many
SELECT *
FROM integration_api_clients
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC;

-- name: RevokeIntegrationAPIClient :one
UPDATE integration_api_clients
SET revoked_at = now()
WHERE id = sqlc.arg(id)
  AND org_id = sqlc.arg(org_id)
RETURNING *;

-- name: TouchIntegrationAPIClientLastUsed :exec
UPDATE integration_api_clients
SET last_used_at = now()
WHERE id = sqlc.arg(id);

-- name: CountSchemesByOrgAndIDs :one
SELECT COUNT(*)::int
FROM schemes
WHERE org_id = sqlc.arg(org_id)
  AND id = ANY(sqlc.arg(scheme_ids)::uuid[]);
