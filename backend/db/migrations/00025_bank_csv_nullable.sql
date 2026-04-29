-- +goose Up
ALTER TABLE bank_statement_imports ALTER COLUMN raw_csv DROP NOT NULL;

-- +goose Down
ALTER TABLE bank_statement_imports ALTER COLUMN raw_csv SET NOT NULL;
