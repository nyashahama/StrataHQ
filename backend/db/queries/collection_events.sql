-- name: ListAttentionAccountsByOrg :many
SELECT
    la.id AS levy_account_id,
    lp.scheme_id,
    s.name AS scheme_name,
    la.unit_id,
    u.identifier AS unit_identifier,
    u.owner_name,
    la.amount_cents,
    la.paid_cents,
    la.due_date,
    la.status
FROM levy_accounts la
JOIN levy_periods lp ON lp.id = la.period_id
JOIN schemes s ON s.id = lp.scheme_id
JOIN units u ON u.id = la.unit_id
WHERE s.org_id = $1
  AND la.amount_cents > la.paid_cents
  AND la.due_date <= CURRENT_DATE
ORDER BY la.due_date ASC, (la.amount_cents - la.paid_cents) DESC;

-- name: ListAttentionAccountsByScheme :many
SELECT
    la.id AS levy_account_id,
    lp.scheme_id,
    s.name AS scheme_name,
    la.unit_id,
    u.identifier AS unit_identifier,
    u.owner_name,
    la.amount_cents,
    la.paid_cents,
    la.due_date,
    la.status
FROM levy_accounts la
JOIN levy_periods lp ON lp.id = la.period_id
JOIN schemes s ON s.id = lp.scheme_id
JOIN units u ON u.id = la.unit_id
WHERE lp.scheme_id = $1
  AND la.amount_cents > la.paid_cents
  AND la.due_date <= CURRENT_DATE
ORDER BY la.due_date ASC, (la.amount_cents - la.paid_cents) DESC;

-- name: ListCollectionEventsByAccountIDs :many
SELECT
    id,
    scheme_id,
    levy_account_id,
    actor_user_id,
    actor_role,
    event_type,
    note,
    promise_amount_cents,
    promise_date,
    email_to,
    email_subject,
    email_body,
    email_status,
    email_error,
    whatsapp_to,
    whatsapp_body,
    whatsapp_status,
    whatsapp_error,
    created_at
FROM collection_events
WHERE levy_account_id = ANY($1::uuid[])
ORDER BY created_at DESC;

-- name: CreateCollectionEvent :one
INSERT INTO collection_events (
    scheme_id,
    levy_account_id,
    actor_user_id,
    actor_role,
    event_type,
    note,
    promise_amount_cents,
    promise_date,
    email_to,
    email_subject,
    email_body,
    email_status,
    email_error,
    whatsapp_to,
    whatsapp_body,
    whatsapp_status,
    whatsapp_error
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13,
    $14, $15, $16, $17
)
RETURNING *;

-- name: MarkCollectionEventEmailDelivery :one
UPDATE collection_events
SET email_status = $2,
    email_error = $3
WHERE id = $1
RETURNING *;

-- name: MarkCollectionEventWhatsAppDelivery :one
UPDATE collection_events
SET whatsapp_status = $2,
    whatsapp_error = $3
WHERE id = $1
RETURNING *;

-- name: GetCollectionEventByID :one
SELECT *
FROM collection_events
WHERE id = $1;