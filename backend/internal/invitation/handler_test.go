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
	revokeFn func(context.Context, string, string) error
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

func (s *stubServicer) Revoke(ctx context.Context, orgID string, invitationID string) error {
	if s.revokeFn == nil {
		panic("unexpected Revoke call")
	}
	return s.revokeFn(ctx, orgID, invitationID)
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

func TestHandlerRevokeReturnsConflictForNonPendingInvitation(t *testing.T) {
	h := NewHandler(&stubServicer{
		revokeFn: func(context.Context, string, string) error {
			return ErrInvalidToken
		},
	}, "http://localhost:3000")

	req := httptest.NewRequest(http.MethodDelete, "/invitations/11111111-1111-1111-1111-111111111111", nil)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", string(auth.RoleAdmin)))
	w := httptest.NewRecorder()

	h.Revoke(w, req)
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
	if payload.Error.Message != "only pending invitations can be revoked" {
		t.Fatalf("message = %q, want %q", payload.Error.Message, "only pending invitations can be revoked")
	}
}

func TestHandlerCreateReturnsConflictForDuplicatePendingInvitation(t *testing.T) {
	h := NewHandler(&stubServicer{
		createFn: func(context.Context, string, CreateParams, string) (*InvitationResponse, error) {
			return nil, ErrDuplicateInvite
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
	if payload.Error.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestHandlerCreateRejectsInvalidEmail(t *testing.T) {
	called := false
	h := NewHandler(&stubServicer{
		createFn: func(context.Context, string, CreateParams, string) (*InvitationResponse, error) {
			called = true
			return nil, nil
		},
	}, "http://localhost:3000")

	body, _ := json.Marshal(map[string]string{
		"email":     "User Example <user@example.com>",
		"full_name": "User Example",
		"role":      "trustee",
		"scheme_id": "11111111-1111-1111-1111-111111111111",
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", string(auth.RoleAdmin)))
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if called {
		t.Fatal("service should not be called for invalid invitation email")
	}
}

func TestHandlerCreateNormalizesEmailAndFullName(t *testing.T) {
	var captured CreateParams
	h := NewHandler(&stubServicer{
		createFn: func(_ context.Context, _ string, p CreateParams, _ string) (*InvitationResponse, error) {
			captured = p
			return &InvitationResponse{ID: "i1", Email: p.Email, FullName: p.FullName, Role: p.Role, SchemeID: p.SchemeID}, nil
		},
	}, "http://localhost:3000")

	body, _ := json.Marshal(map[string]string{
		"email":     " Trustee@Example.COM ",
		"full_name": " Trustee User ",
		"role":      "trustee",
		"scheme_id": "11111111-1111-1111-1111-111111111111",
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", string(auth.RoleAdmin)))
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if captured.Email != "trustee@example.com" {
		t.Fatalf("email = %q, want trustee@example.com", captured.Email)
	}
	if captured.FullName != "Trustee User" {
		t.Fatalf("fullName = %q, want Trustee User", captured.FullName)
	}
}

func uuidMustString(raw string) string {
	return raw
}
