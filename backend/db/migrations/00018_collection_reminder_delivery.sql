-- +goose Up

ALTER TABLE collection_events
    ADD COLUMN email_to TEXT,
    ADD COLUMN email_subject TEXT,
    ADD COLUMN email_body TEXT,
    ADD COLUMN email_status TEXT CHECK (email_status IN ('sent', 'failed', 'skipped') OR email_status IS NULL),
    ADD COLUMN email_error TEXT,
    ADD COLUMN whatsapp_to TEXT,
    ADD COLUMN whatsapp_body TEXT,
    ADD COLUMN whatsapp_status TEXT CHECK (whatsapp_status IN ('sent', 'failed', 'skipped') OR whatsapp_status IS NULL),
    ADD COLUMN whatsapp_error TEXT;

-- +goose Down

ALTER TABLE collection_events
    DROP COLUMN IF EXISTS whatsapp_error,
    DROP COLUMN IF EXISTS whatsapp_status,
    DROP COLUMN IF EXISTS whatsapp_body,
    DROP COLUMN IF EXISTS whatsapp_to,
    DROP COLUMN IF EXISTS email_error,
    DROP COLUMN IF EXISTS email_status,
    DROP COLUMN IF EXISTS email_body,
    DROP COLUMN IF EXISTS email_subject,
    DROP COLUMN IF EXISTS email_to;