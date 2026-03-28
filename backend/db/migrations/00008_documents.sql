-- +goose Up

CREATE TYPE document_file_type AS ENUM ('pdf', 'docx', 'xlsx', 'jpg', 'png');

CREATE TYPE document_category AS ENUM ('rules', 'minutes', 'insurance', 'financial', 'other');

CREATE TABLE scheme_documents (
    id                  UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id           UUID                 NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    name                TEXT                 NOT NULL,
    storage_key         TEXT                 NOT NULL,
    file_type           document_file_type   NOT NULL,
    category            document_category    NOT NULL,
    size_bytes          BIGINT               NOT NULL CHECK (size_bytes >= 0),
    uploaded_by_user_id UUID                 REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scheme_documents_scheme_id ON scheme_documents(scheme_id);

-- +goose Down

DROP INDEX IF EXISTS idx_scheme_documents_scheme_id;
DROP TABLE IF EXISTS scheme_documents;
DROP TYPE  IF EXISTS document_category;
DROP TYPE  IF EXISTS document_file_type;
