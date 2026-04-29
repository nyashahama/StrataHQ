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

-- name: GetConnectedWhatsAppThreadByPhone :many
SELECT * FROM whatsapp_threads
WHERE phone_number = $1
  AND connected = TRUE
ORDER BY last_active_at DESC;

-- name: IncrementWhatsAppThreadUnread :exec
UPDATE whatsapp_threads
SET unread_count = unread_count + 1,
    last_active_at = NOW()
WHERE id = $1;

-- name: ConnectWhatsAppThread :exec
UPDATE whatsapp_threads
SET connected = TRUE,
    consented_at = NOW(),
    phone_number = $2,
    last_active_at = NOW()
WHERE id = $1;

-- name: CreateWhatsAppMessageMedia :one
INSERT INTO whatsapp_message_media (
    message_id,
    provider,
    provider_media_sid,
    media_url,
    content_type
)
VALUES (
    sqlc.arg(message_id),
    sqlc.arg(provider),
    sqlc.arg(provider_media_sid),
    sqlc.arg(media_url),
    sqlc.arg(content_type)
)
ON CONFLICT (message_id, provider_media_sid) WHERE provider_media_sid IS NOT NULL
DO UPDATE SET
    media_url = EXCLUDED.media_url,
    content_type = EXCLUDED.content_type
RETURNING *;

-- name: ListWhatsAppMessageMediaByThread :many
SELECT wmm.*
FROM whatsapp_message_media wmm
JOIN whatsapp_messages wm ON wm.id = wmm.message_id
WHERE wm.thread_id = sqlc.arg(thread_id)
ORDER BY wmm.created_at ASC;

-- name: CountWhatsAppMessageMediaByMessage :one
SELECT COUNT(*)
FROM whatsapp_message_media
WHERE message_id = sqlc.arg(message_id);

-- name: UpdateWhatsAppMessageMaintenanceRequest :exec
UPDATE whatsapp_messages
SET maintenance_request_id = sqlc.arg(maintenance_request_id)
WHERE id = sqlc.arg(id);

-- name: GetWhatsAppMessageWithThread :one
SELECT
    wm.*,
    wt.scheme_id,
    wt.unit_id,
    wt.resident_user_id,
    wt.phone_number
FROM whatsapp_messages wm
JOIN whatsapp_threads wt ON wt.id = wm.thread_id
WHERE wm.id = sqlc.arg(message_id)
LIMIT 1;

-- name: CreateWhatsAppMaintenanceIntake :one
INSERT INTO whatsapp_maintenance_intakes (
    scheme_id,
    thread_id,
    message_id,
    unit_id,
    maintenance_request_id,
    status,
    category,
    title,
    description,
    media_count
)
VALUES (
    sqlc.arg(scheme_id),
    sqlc.arg(thread_id),
    sqlc.arg(message_id),
    sqlc.arg(unit_id),
    sqlc.arg(maintenance_request_id),
    sqlc.arg(status),
    sqlc.arg(category),
    sqlc.arg(title),
    sqlc.arg(description),
    sqlc.arg(media_count)
)
ON CONFLICT (message_id)
DO UPDATE SET
    maintenance_request_id = COALESCE(EXCLUDED.maintenance_request_id, whatsapp_maintenance_intakes.maintenance_request_id),
    status = EXCLUDED.status,
    category = EXCLUDED.category,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    media_count = EXCLUDED.media_count
RETURNING *;

-- name: ListWhatsAppMaintenanceIntakesByScheme :many
SELECT
    wmi.*,
    u.identifier AS unit_identifier,
    u.owner_name
FROM whatsapp_maintenance_intakes wmi
JOIN units u ON u.id = wmi.unit_id
WHERE wmi.scheme_id = sqlc.arg(scheme_id)
ORDER BY wmi.created_at DESC;

-- name: GetWhatsAppMaintenanceIntake :one
SELECT *
FROM whatsapp_maintenance_intakes
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: DismissWhatsAppMaintenanceIntake :one
UPDATE whatsapp_maintenance_intakes
SET status = 'dismissed'
WHERE id = sqlc.arg(id)
RETURNING *;
