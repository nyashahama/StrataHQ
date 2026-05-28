-- +goose Up

-- Step 1: Add permissive policies on every table that has RLS enabled
-- but no policies yet. This ensures data remains accessible after FORCE
-- ROW LEVEL SECURITY is applied in step 2.
-- +goose StatementBegin
DO $$
DECLARE
    row_record RECORD;
BEGIN
    FOR row_record IN
        SELECT t.schemaname, t.tablename
        FROM pg_tables t
        JOIN pg_class c ON c.relname = t.tablename AND c.relnamespace = (
            SELECT oid FROM pg_namespace WHERE nspname = t.schemaname
        )
        WHERE t.schemaname = 'public'
          AND t.tablename <> 'goose_db_version'
          AND c.relrowsecurity = true
          AND NOT EXISTS (
              SELECT 1 FROM pg_policy p
              WHERE p.polrelid = c.oid
          )
    LOOP
        EXECUTE format(
            'CREATE POLICY rls_default ON %I.%I FOR ALL USING (true) WITH CHECK (true)',
            row_record.schemaname,
            row_record.tablename
        );
    END LOOP;
END $$;
-- +goose StatementEnd

-- Step 2: Force RLS on all public tables where it is enabled.
-- This ensures even the table owner (the application's database role)
-- is subject to RLS policies.
-- +goose StatementBegin
DO $$
DECLARE
    row_record RECORD;
BEGIN
    FOR row_record IN
        SELECT t.schemaname, t.tablename
        FROM pg_tables t
        JOIN pg_class c ON c.relname = t.tablename AND c.relnamespace = (
            SELECT oid FROM pg_namespace WHERE nspname = t.schemaname
        )
        WHERE t.schemaname = 'public'
          AND t.tablename <> 'goose_db_version'
          AND c.relrowsecurity = true
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY',
            row_record.schemaname,
            row_record.tablename
        );
    END LOOP;
END $$;
-- +goose StatementEnd


-- +goose Down

-- Remove forced RLS and default policies
-- +goose StatementBegin
DO $$
DECLARE
    row_record RECORD;
BEGIN
    FOR row_record IN
        SELECT t.schemaname, t.tablename
        FROM pg_tables t
        JOIN pg_class c ON c.relname = t.tablename AND c.relnamespace = (
            SELECT oid FROM pg_namespace WHERE nspname = t.schemaname
        )
        WHERE t.schemaname = 'public'
          AND t.tablename <> 'goose_db_version'
          AND c.relrowsecurity = true
          AND c.relforcerowsecurity = true
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.%I NO FORCE ROW LEVEL SECURITY',
            row_record.schemaname,
            row_record.tablename
        );
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    row_record RECORD;
BEGIN
    FOR row_record IN
        SELECT p.polname, t.schemaname, t.tablename
        FROM pg_policy p
        JOIN pg_class c ON c.oid = p.polrelid
        JOIN pg_tables t ON t.tablename = c.relname
            AND t.schemaname = (
                SELECT nspname FROM pg_namespace WHERE oid = c.relnamespace
            )
        WHERE p.polname = 'rls_default'
          AND t.schemaname = 'public'
    LOOP
        EXECUTE format(
            'DROP POLICY IF EXISTS %I ON %I.%I',
            row_record.polname,
            row_record.schemaname,
            row_record.tablename
        );
    END LOOP;
END $$;
-- +goose StatementEnd
