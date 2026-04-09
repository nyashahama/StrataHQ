package invitation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/auth"
)

type stubServicer struct {
	createFn func(ctx context.Context, orgID string, p CreateParams, appBaseURL string) (*InvitationResponse, error)
}

func (s *stubServicer) Create(ctx context.Context, orgID string, p CreateParams, appBaseURL string) (*InvitationResponse, error) {
	return s.createFn(ctx, orgID, p, appBaseURL)
}

func (s *stubServicer) List(context.Context, string) ([]InvitationResponse, error) {
	panic("unexpected List call")
}

func (s *stubServicer) Resend(context.Context, string, string, string) (*InvitationResponse, error) {
	panic("unexpected Resend call")
}

func (s *stubServicer) Revoke(context.Context, string, string) error {
	panic("unexpected Revoke call")
}

func (s *stubServicer) Verify(context.Context, string) (*VerifyResponse, error) {
	panic("unexpected Verify call")
}

func (s *stubServicer) Accept(context.Context, string, string) (*auth.AuthResponse, error) {
	panic("unexpected Accept call")
}

func TestHandlerCreateReturnsForbiddenForForeignScheme(t *testing.T) {
	h := NewHandler(&stubServicer{
		createFn: func(context.Context, string, CreateParams, string) (*InvitationResponse, error) {
			return nil, ErrForbidden
		},
	}, "http://localhost:3000")

	body, _ := json.Marshal(map[string]string{
		"email":     "user@example.com",
		"full_name": "User Example",
		"role":      "trustee",
		"scheme_id": "11111111-1111-1111-1111-111111111111",
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), uuidMustString("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), uuidMustString("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), string(auth.RoleAdmin)))
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func uuidMustString(raw string) string {
	return raw
}
