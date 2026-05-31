-- name: CreateEarlyAccessRequest :one
INSERT INTO early_access_requests (
    full_name, email, scheme_name, unit_count
)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListEarlyAccessRequests :many
SELECT *
FROM early_access_requests
ORDER BY created_at DESC;

-- name: GetEarlyAccessRequest :one
SELECT *
FROM early_access_requests
WHERE id = $1;

-- name: UpdateEarlyAccessStatus :one
UPDATE early_access_requests
SET status = $2, reviewed_at = NOW()
WHERE id = $1
  AND status = 'pending'
RETURNING *;
