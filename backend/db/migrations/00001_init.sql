-- +goose Up
-- Placeholder init migration.
-- Domain-specific tables will be added as the project is implemented.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS schema_migrations_lock (
    id INTEGER PRIMARY KEY DEFAULT 1,
    locked BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down

DROP TABLE IF EXISTS schema_migrations_lock;
