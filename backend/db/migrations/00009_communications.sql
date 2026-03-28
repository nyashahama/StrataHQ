-- +goose Up

CREATE TYPE notice_type AS ENUM ('general', 'urgent', 'agm', 'levy');

CREATE TABLE notices (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id         UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    title             TEXT        NOT NULL,
    body              TEXT        NOT NULL,
    type              notice_type NOT NULL DEFAULT 'general',
    sent_by_user_id   UUID        REFERENCES users(id) ON DELETE SET NULL,
    sent_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notices_scheme_id ON notices(scheme_id);
CREATE INDEX idx_notices_sent_at   ON notices(sent_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_notices_sent_at;
DROP INDEX IF EXISTS idx_notices_scheme_id;
DROP TABLE IF EXISTS notices;
DROP TYPE  IF EXISTS notice_type;
