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
	resendFn func(context.Context, string, string, string) (*InvitationResponse, error)
}

func (s *stubServicer) Create(ctx context.Context, orgID string, p CreateParams, appBaseURL string) (*InvitationResponse, error) {
	return s.createFn(ctx, orgID, p, appBaseURL)
}

func (s *stubServicer) List(context.Context, string) ([]InvitationResponse, error) {
	panic("unexpected List call")
}

func (s *stubServicer) Resend(ctx context.Context, orgID string, invitationID string, appBaseURL string) (*InvitationResponse, error) {
	if s.resendFn == nil {
		panic("unexpected Resend call")
	}
	return s.resendFn(ctx, orgID, invitationID, appBaseURL)
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

func TestHandlerResendReturnsConflictForNonPendingInvitation(t *testing.T) {
	h := NewHandler(&stubServicer{
		resendFn: func(context.Context, string, string, string) (*InvitationResponse, error) {
			return nil, ErrInvalidToken
		},
	}, "http://localhost:3000")

	req := httptest.NewRequest(http.MethodPost, "/invitations/11111111-1111-1111-1111-111111111111/resend", nil)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", string(auth.RoleAdmin)))
	w := httptest.NewRecorder()

	h.Resend(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}

	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("response decode: %v", err)
	}
	if payload.Error.Message != "only pending invitations can be resent" {
		t.Fatalf("message = %q, want %q", payload.Error.Message, "only pending invitations can be resent")
	}
}

func uuidMustString(raw string) string {
	return raw
}
