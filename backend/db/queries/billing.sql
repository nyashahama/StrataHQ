-- name: GetOrgSubscription :one
SELECT * FROM org_subscriptions
WHERE org_id = $1
LIMIT 1;

-- name: GetOrgSubscriptionByCustomerID :one
SELECT * FROM org_subscriptions
WHERE customer_id = $1
LIMIT 1;

-- name: GetOrgSubscriptionBySubscriptionID :one
SELECT * FROM org_subscriptions
WHERE subscription_id = $1
LIMIT 1;

-- name: UpsertOrgSubscription :one
INSERT INTO org_subscriptions (
    org_id,
    provider,
    status,
    plan_code,
    customer_id,
    subscription_id,
    checkout_session_id,
    current_period_end,
    cancel_at_period_end
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (org_id)
DO UPDATE SET
    provider             = EXCLUDED.provider,
    status               = EXCLUDED.status,
    plan_code            = EXCLUDED.plan_code,
    customer_id          = EXCLUDED.customer_id,
    subscription_id      = EXCLUDED.subscription_id,
    checkout_session_id  = EXCLUDED.checkout_session_id,
    current_period_end   = EXCLUDED.current_period_end,
    cancel_at_period_end = EXCLUDED.cancel_at_period_end
RETURNING *;
