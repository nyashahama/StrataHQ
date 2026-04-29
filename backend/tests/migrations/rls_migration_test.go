package migrations_test

import (
	"os"
	"strings"
	"testing"
)

func TestSupabaseRLSHardeningMigrationExists(t *testing.T) {
	path := "../../db/migrations/00019_supabase_rls_hardening.sql"

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(content)
	required := []string{
		"ENABLE ROW LEVEL SECURITY",
		"REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated;",
		"REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon, authenticated;",
		"ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON TABLES FROM anon, authenticated;",
	}

	for _, token := range required {
		if !strings.Contains(sql, token) {
			t.Fatalf("missing expected SQL token: %q", token)
		}
	}
}
