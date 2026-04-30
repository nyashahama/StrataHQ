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
		"IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon')",
		"IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated')",
		"IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'postgres')",
	}

	for _, token := range required {
		if !strings.Contains(sql, token) {
			t.Fatalf("missing expected SQL token: %q", token)
		}
	}
}

func TestWhatsAppMaintenanceInboxRLS(t *testing.T) {
	path := "../../db/migrations/00027_whatsapp_maintenance_inbox.sql"

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(content)

	required := []string{
		"CREATE TABLE whatsapp_message_media",
		"ALTER TABLE whatsapp_message_media ENABLE ROW LEVEL SECURITY",
		"CREATE TABLE whatsapp_maintenance_intakes",
		"ALTER TABLE whatsapp_maintenance_intakes ENABLE ROW LEVEL SECURITY",
	}

	for _, token := range required {
		if !strings.Contains(sql, token) {
			t.Fatalf("missing expected SQL token: %q", token)
		}
	}
}
