-- +goose Up

CREATE TABLE org_subscriptions (
    org_id                 UUID        PRIMARY KEY REFERENCES orgs(id) ON DELETE CASCADE,
    provider               TEXT        NOT NULL DEFAULT 'stripe',
    status                 TEXT        NOT NULL DEFAULT 'inactive',
    plan_code              TEXT        NOT NULL DEFAULT 'starter',
    customer_id            TEXT,
    subscription_id        TEXT,
    checkout_session_id    TEXT,
    current_period_end     TIMESTAMPTZ,
    cancel_at_period_end   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER org_subscriptions_set_updated_at
    BEFORE UPDATE ON org_subscriptions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE UNIQUE INDEX idx_org_subscriptions_customer_id
    ON org_subscriptions(customer_id)
    WHERE customer_id IS NOT NULL;

CREATE UNIQUE INDEX idx_org_subscriptions_subscription_id
    ON org_subscriptions(subscription_id)
    WHERE subscription_id IS NOT NULL;

-- +goose Down

DROP INDEX   IF EXISTS idx_org_subscriptions_subscription_id;
DROP INDEX   IF EXISTS idx_org_subscriptions_customer_id;
DROP TRIGGER IF EXISTS org_subscriptions_set_updated_at ON org_subscriptions;
DROP TABLE   IF EXISTS org_subscriptions;
