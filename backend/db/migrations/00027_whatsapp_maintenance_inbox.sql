-- +goose Up

CREATE TABLE whatsapp_message_media (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id         UUID        NOT NULL REFERENCES whatsapp_messages(id) ON DELETE CASCADE,
    provider           TEXT        NOT NULL DEFAULT 'twilio',
    provider_media_sid TEXT,
    media_url          TEXT        NOT NULL,
    content_type       TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE whatsapp_message_media ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_whatsapp_message_media_message_id
    ON whatsapp_message_media (message_id);

CREATE UNIQUE INDEX idx_whatsapp_message_media_message_provider_sid
    ON whatsapp_message_media (message_id, provider_media_sid)
    WHERE provider_media_sid IS NOT NULL;

CREATE TABLE whatsapp_maintenance_intakes (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id              UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    thread_id              UUID        NOT NULL REFERENCES whatsapp_threads(id) ON DELETE CASCADE,
    message_id             UUID        NOT NULL UNIQUE REFERENCES whatsapp_messages(id) ON DELETE CASCADE,
    unit_id                UUID        NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    maintenance_request_id UUID        REFERENCES maintenance_requests(id) ON DELETE SET NULL,
    status                 TEXT        NOT NULL CHECK (status IN ('candidate', 'ticket_created', 'dismissed')),
    category               maintenance_category NOT NULL DEFAULT 'other',
    title                  TEXT        NOT NULL,
    description            TEXT        NOT NULL,
    media_count            INTEGER     NOT NULL DEFAULT 0 CHECK (media_count >= 0),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER whatsapp_maintenance_intakes_set_updated_at
    BEFORE UPDATE ON whatsapp_maintenance_intakes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE whatsapp_maintenance_intakes ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_whatsapp_maintenance_intakes_scheme_status_created
    ON whatsapp_maintenance_intakes (scheme_id, status, created_at DESC);

CREATE INDEX idx_whatsapp_maintenance_intakes_request
    ON whatsapp_maintenance_intakes (maintenance_request_id)
    WHERE maintenance_request_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_whatsapp_maintenance_intakes_request;
DROP INDEX IF EXISTS idx_whatsapp_maintenance_intakes_scheme_status_created;
DROP TRIGGER IF EXISTS whatsapp_maintenance_intakes_set_updated_at ON whatsapp_maintenance_intakes;
DROP TABLE IF EXISTS whatsapp_maintenance_intakes;
DROP INDEX IF EXISTS idx_whatsapp_message_media_message_provider_sid;
DROP INDEX IF EXISTS idx_whatsapp_message_media_message_id;
DROP TABLE IF EXISTS whatsapp_message_media;
