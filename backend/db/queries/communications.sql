-- name: CreateNotice :one
INSERT INTO notices (scheme_id, title, body, type, sent_by_user_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetNotice :one
SELECT * FROM notices
WHERE id = $1
LIMIT 1;

-- name: ListNoticesByScheme :many
SELECT * FROM notices
WHERE scheme_id = $1
ORDER BY sent_at DESC;

-- name: ListNoticesBySchemeAndType :many
SELECT * FROM notices
WHERE scheme_id = $1 AND type = $2
ORDER BY sent_at DESC;
