-- +goose Up

CREATE TYPE document_visibility AS ENUM ('all', 'trustee', 'admin');

ALTER TABLE scheme_documents
    ADD COLUMN visibility document_visibility NOT NULL DEFAULT 'all';

-- +goose Down

ALTER TABLE scheme_documents
    DROP COLUMN IF EXISTS visibility;

DROP TYPE IF EXISTS document_visibility;
