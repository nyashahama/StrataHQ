-- +goose Up

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone TEXT;

ALTER TABLE orgs
    ADD COLUMN IF NOT EXISTS contact_phone TEXT;

-- +goose Down

ALTER TABLE orgs
    DROP COLUMN IF EXISTS contact_phone;

ALTER TABLE users
    DROP COLUMN IF EXISTS phone;
