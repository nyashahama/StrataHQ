package integrations

import (
	"encoding/json"
	"testing"
)

func TestOpenAPIDocumentIncludesImplementedPaths(t *testing.T) {
	var doc struct {
		OpenAPI string         `json:"openapi"`
		Paths   map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(openAPIDocument, &doc); err != nil {
		t.Fatalf("openapi document is invalid json: %v", err)
	}
	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("openapi version = %q", doc.OpenAPI)
	}
	required := []string{
		"/schemes",
		"/schemes/{schemeId}",
		"/schemes/{schemeId}/units",
		"/schemes/{schemeId}/levy-periods",
		"/schemes/{schemeId}/levy-accounts",
		"/schemes/{schemeId}/levy-payments",
		"/schemes/{schemeId}/financials",
	}
	for _, path := range required {
		if _, ok := doc.Paths[path]; !ok {
			t.Fatalf("missing path %s", path)
		}
	}
}
