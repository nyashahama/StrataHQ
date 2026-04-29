-- +goose Up

-- Enable RLS on all current public tables, including those created after
-- migration 00019 (resource_audit_events, background_jobs, bank_statement_imports,
-- bank_statement_rows). There is no ALTER DEFAULT for RLS, so follow-up
-- migrations must re-run this block.

-- +goose StatementBegin
DO $$
DECLARE
    row_record RECORD;
BEGIN
    FOR row_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename <> 'goose_db_version'
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY',
            row_record.schemaname,
            row_record.tablename
        );
    END LOOP;
END $$;
-- +goose StatementEnd

-- Revoke direct table access from the Supabase roles on all current tables.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        EXECUTE 'REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon';
        EXECUTE 'REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon';
    END IF;

    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        EXECUTE 'REVOKE ALL ON ALL TABLES IN SCHEMA public FROM authenticated';
        EXECUTE 'REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM authenticated';
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down

-- Disable RLS on all public tables (reverse of the Up block).
-- +goose StatementBegin
DO $$
DECLARE
    row_record RECORD;
BEGIN
    FOR row_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename <> 'goose_db_version'
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY',
            row_record.schemaname,
            row_record.tablename
        );
    END LOOP;
END $$;
-- +goose StatementEnd
