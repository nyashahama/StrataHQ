package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/auth"
)

func TestListSchemeEventsRejectsInvalidLimits(t *testing.T) {
	tests := []string{
		"-1",
		"0",
		"201",
		"4294967296",
	}
	for _, limit := range tests {
		t.Run(limit, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/audit/schemes/s1/events?limit="+limit, nil)
			req = req.WithContext(auth.ContextWithIdentity(req.Context(), "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", string(auth.RoleAdmin)))
			w := httptest.NewRecorder()

			NewHandler(&ResourceService{}).ListSchemeEvents(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}
