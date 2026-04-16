-- +goose Up

CREATE TABLE collection_events (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id            UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    levy_account_id      UUID        NOT NULL REFERENCES levy_accounts(id) ON DELETE CASCADE,
    actor_user_id        UUID        REFERENCES users(id) ON DELETE SET NULL,
    actor_role           TEXT        NOT NULL DEFAULT '',
    event_type           TEXT        NOT NULL CHECK (
        event_type IN (
            'reminder_sent',
            'follow_up_logged',
            'promise_to_pay',
            'legal_review_flagged'
        )
    ),
    note                 TEXT,
    promise_amount_cents BIGINT,
    promise_date         DATE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_collection_events_account_created_at
    ON collection_events (levy_account_id, created_at DESC);
CREATE INDEX idx_collection_events_scheme_created_at
    ON collection_events (scheme_id, created_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_collection_events_scheme_created_at;
DROP INDEX IF EXISTS idx_collection_events_account_created_at;
DROP TABLE IF EXISTS collection_events;