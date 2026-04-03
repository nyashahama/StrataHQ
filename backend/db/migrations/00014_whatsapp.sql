-- +goose Up

CREATE TYPE whatsapp_message_sender AS ENUM ('resident', 'bot', 'operator');
CREATE TYPE whatsapp_broadcast_type AS ENUM ('levy', 'agm', 'maintenance', 'general');

CREATE TABLE whatsapp_threads (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id        UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    unit_id          UUID        NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    resident_user_id UUID        REFERENCES users(id) ON DELETE SET NULL,
    phone_number     TEXT,
    connected        BOOLEAN     NOT NULL DEFAULT FALSE,
    consented_at     TIMESTAMPTZ,
    unread_count     INTEGER     NOT NULL DEFAULT 0,
    last_active_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT whatsapp_threads_scheme_unit_unique UNIQUE (scheme_id, unit_id)
);

CREATE TRIGGER whatsapp_threads_set_updated_at
    BEFORE UPDATE ON whatsapp_threads
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE whatsapp_messages (
    id                     UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id              UUID                     NOT NULL REFERENCES whatsapp_threads(id) ON DELETE CASCADE,
    sender                 whatsapp_message_sender  NOT NULL,
    body                   TEXT                     NOT NULL,
    maintenance_request_id UUID                     REFERENCES maintenance_requests(id) ON DELETE SET NULL,
    notice_id              UUID                     REFERENCES notices(id) ON DELETE SET NULL,
    created_at             TIMESTAMPTZ              NOT NULL DEFAULT NOW()
);

CREATE TABLE whatsapp_broadcasts (
    id              UUID                    PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id       UUID                    NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    sent_by_user_id UUID                    REFERENCES users(id) ON DELETE SET NULL,
    type            whatsapp_broadcast_type NOT NULL DEFAULT 'general',
    message         TEXT                    NOT NULL,
    recipient_count INTEGER                 NOT NULL DEFAULT 0,
    sent_at         TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_whatsapp_threads_scheme_id ON whatsapp_threads(scheme_id);
CREATE INDEX idx_whatsapp_threads_last_active_at ON whatsapp_threads(last_active_at DESC);
CREATE INDEX idx_whatsapp_messages_thread_id ON whatsapp_messages(thread_id, created_at DESC);
CREATE INDEX idx_whatsapp_broadcasts_scheme_id ON whatsapp_broadcasts(scheme_id, sent_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_whatsapp_broadcasts_scheme_id;
DROP INDEX IF EXISTS idx_whatsapp_messages_thread_id;
DROP INDEX IF EXISTS idx_whatsapp_threads_last_active_at;
DROP INDEX IF EXISTS idx_whatsapp_threads_scheme_id;
DROP TABLE IF EXISTS whatsapp_broadcasts;
DROP TABLE IF EXISTS whatsapp_messages;
DROP TRIGGER IF EXISTS whatsapp_threads_set_updated_at ON whatsapp_threads;
DROP TABLE IF EXISTS whatsapp_threads;
DROP TYPE IF EXISTS whatsapp_broadcast_type;
DROP TYPE IF EXISTS whatsapp_message_sender;
