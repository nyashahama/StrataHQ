-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Trigger function: keep updated_at current on any UPDATE.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- +goose Down
DROP FUNCTION IF EXISTS set_updated_at();
