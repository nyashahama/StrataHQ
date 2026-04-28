package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemeScopedServicesResolveAccessBeforeReturningData(t *testing.T) {
	repoRoot := findRepoRoot(t)
	required := map[string]string{
		"internal/ai/service.go":             "resolveSchemeAccess",
		"internal/communications/service.go": "resolveAccess",
		"internal/compliance/service.go":     "resolveSchemeAccess",
		"internal/documents/service.go":      "resolveAccess",
		"internal/financials/service.go":     "resolveSchemeAccess",
		"internal/maintenance/service.go":    "resolveAccess",
		"internal/scheme/service.go":         "resolveSchemeAccess",
	}

	for rel, want := range required {
		body, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if !strings.Contains(string(body), want) {
			t.Fatalf("%s must contain %s to enforce scheme access", rel, want)
		}
		if !strings.Contains(string(body), "ErrForbidden") {
			t.Fatalf("%s must return ErrForbidden for denied access", rel)
		}
	}
}

func TestWriteOperationsCheckPrivilegedRoles(t *testing.T) {
	repoRoot := findRepoRoot(t)
	requiredSnippets := map[string][]string{
		"internal/documents/service.go": {
			"IsAdminRole(access.role)",
			"return nil, ErrForbidden",
		},
		"internal/communications/service.go": {
			"IsResidentRole(access.role)",
			"return nil, ErrForbidden",
		},
		"internal/financials/service.go": {
			"role == string(auth.RoleResident)",
			"return nil, ErrForbidden",
		},
		"internal/maintenance/service.go": {
			"IsResidentRole(access.role)",
			"return nil, ErrForbidden",
		},
		"internal/scheme/service.go": {
			"IsAdminRole(role)",
			"return nil, ErrForbidden",
		},
	}

	for rel, snippets := range requiredSnippets {
		body, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(body)
		for _, snippet := range snippets {
			if !strings.Contains(text, snippet) {
				t.Fatalf("%s missing required authorization snippet %q", rel, snippet)
			}
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find backend go.mod")
		}
		dir = parent
	}
}
