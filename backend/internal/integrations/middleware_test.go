package integrations

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIntegrationIdentityContext(t *testing.T) {
	identity := Identity{
		ClientID:  "client-1",
		OrgID:     "org-1",
		Scopes:    []string{"read:schemes"},
		SchemeIDs: []string{"scheme-1"},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	withIdentity := req.WithContext(ContextWithIdentity(req.Context(), identity))

	got, ok := IdentityFromRequest(withIdentity)
	if !ok {
		t.Fatal("expected identity")
	}
	if got.ClientID != "client-1" || !got.HasScope("read:schemes") || !got.CanAccessScheme("scheme-1") {
		t.Fatalf("unexpected identity: %+v", got)
	}
	if got.CanAccessScheme("scheme-2") {
		t.Fatalf("identity should not access scheme-2")
	}
}
