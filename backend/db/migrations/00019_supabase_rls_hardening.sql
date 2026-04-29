-- +goose Up

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

REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon, authenticated;

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON TABLES FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON SEQUENCES FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM anon, authenticated;

-- +goose Down

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
