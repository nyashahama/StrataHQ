-- name: CreateWhatsAppThread :one
INSERT INTO whatsapp_threads (
    scheme_id, unit_id, resident_user_id, phone_number, connected, consented_at, unread_count, last_active_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetWhatsAppThreadBySchemeAndUnit :one
SELECT * FROM whatsapp_threads
WHERE scheme_id = $1 AND unit_id = $2
LIMIT 1;

-- name: ListWhatsAppThreadsDetailedByScheme :many
SELECT wt.*, u.identifier AS unit_identifier, COALESCE(res.full_name, u.owner_name) AS owner_name
FROM whatsapp_threads wt
JOIN units u ON u.id = wt.unit_id
LEFT JOIN users res ON res.id = wt.resident_user_id
WHERE wt.scheme_id = $1
ORDER BY wt.last_active_at DESC, u.identifier ASC;

-- name: CountConnectedWhatsAppThreadsByScheme :one
SELECT COUNT(*)
FROM whatsapp_threads
WHERE scheme_id = $1
  AND connected = TRUE;

-- name: CreateWhatsAppMessage :one
INSERT INTO whatsapp_messages (
    thread_id, sender, body, maintenance_request_id, notice_id
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListWhatsAppMessagesByThread :many
SELECT *
FROM whatsapp_messages
WHERE thread_id = $1
ORDER BY created_at ASC;

-- name: TouchWhatsAppThread :exec
UPDATE whatsapp_threads
SET unread_count = $2,
    last_active_at = $3
WHERE id = $1;

-- name: CreateWhatsAppBroadcast :one
INSERT INTO whatsapp_broadcasts (
    scheme_id, sent_by_user_id, type, message, recipient_count
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListWhatsAppBroadcastsDetailedByScheme :many
SELECT wb.*, u.full_name AS sent_by_name
FROM whatsapp_broadcasts wb
LEFT JOIN users u ON u.id = wb.sent_by_user_id
WHERE wb.scheme_id = $1
ORDER BY wb.sent_at DESC;
