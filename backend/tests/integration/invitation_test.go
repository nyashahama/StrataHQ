//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/invitation"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/platform/database"
)

func newInvitationHandler(t *testing.T) (*invitation.Handler, *notification.NoopSender) {
	t.Helper()
	pool := &database.Pool{Pool: testDB, Q: dbgen.New(testDB)}
	sender := &notification.NoopSender{}
	svc := invitation.NewService(pool, sender, testJWTSigningKey, 15*time.Minute, 7*24*time.Hour)
	return invitation.NewHandler(svc, "http://localhost:3000"), sender
}

func setupAgent(t *testing.T) (accessToken, orgID string) {
	t.Helper()
	h := newAuthHandler(t)
	email := uniqueEmail(t)
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Test Agent",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setupAgent register: %d %s", w.Code, w.Body)
	}
	var resp struct{ Data auth.AuthResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&resp)
	// Extract orgID from JWT claims
	claims, err := auth.ValidateAccessToken(resp.Data.AccessToken, testJWTSigningKey)
	if err != nil {
		t.Fatalf("setupAgent: invalid token: %v", err)
	}
	return resp.Data.AccessToken, claims.OrgID
}

func setupScheme(t *testing.T, accessToken string) string {
	t.Helper()
	h := newAuthHandler(t)
	body, _ := json.Marshal(map[string]any{
		"org_name": "Test Org", "contact_email": "admin@test.co.za",
		"scheme_name": "Test Scheme", "scheme_address": "1 Test St",
		"unit_count": 5,
	})
	req := httptest.NewRequest(http.MethodPost, "/onboarding/setup", bytes.NewReader(body))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setupScheme: %d %s", w.Code, w.Body)
	}
	var resp struct{ Data auth.SetupResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.Data.Scheme.ID
}

func withOrgRoleContext(r *http.Request, orgID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), auth.OrgIDKey, orgID)
	ctx = context.WithValue(ctx, auth.RoleKey, role)
	return r.WithContext(ctx)
}

func TestInvitation_CreateListRevokeResend(t *testing.T) {
	ih, sender := newInvitationHandler(t)
	agentToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, agentToken)

	// Create invitation
	body, _ := json.Marshal(map[string]string{
		"email": "trustee@example.com", "full_name": "Jane Trustee",
		"role": "trustee", "scheme_id": schemeID,
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req = withOrgRoleContext(req, orgID, "admin")
	w := httptest.NewRecorder()
	ih.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	if len(sender.InvitationsSent) != 1 || sender.InvitationsSent[0] != "trustee@example.com" {
		t.Errorf("expected 1 invitation email to trustee@example.com, got %v", sender.InvitationsSent)
	}
	var invResp struct{ Data invitation.InvitationResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&invResp)
	invID := invResp.Data.ID
	if invID == "" {
		t.Fatal("create: missing invitation id")
	}

	// List — 1 pending
	req = httptest.NewRequest(http.MethodGet, "/invitations", nil)
	req = withOrgRoleContext(req, orgID, "admin")
	w = httptest.NewRecorder()
	ih.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	var listResp struct{ Data []invitation.InvitationResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Data) != 1 {
		t.Errorf("list: expected 1, got %d", len(listResp.Data))
	}

	// Resend — sends a 2nd email
	req = httptest.NewRequest(http.MethodPost, "/invitations/"+invID+"/resend", nil)
	req = withOrgRoleContext(req, orgID, "admin")
	// chi URL params must be set manually in unit tests
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", invID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.OrgIDKey, orgID)
	ctx = context.WithValue(ctx, auth.RoleKey, "admin")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	ih.Resend(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resend: %d %s", w.Code, w.Body)
	}
	if len(sender.InvitationsSent) != 2 {
		t.Errorf("resend: expected 2 emails sent, got %d", len(sender.InvitationsSent))
	}

	// Revoke
	req = httptest.NewRequest(http.MethodDelete, "/invitations/"+invID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", invID)
	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.OrgIDKey, orgID)
	ctx = context.WithValue(ctx, auth.RoleKey, "admin")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	ih.Revoke(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", w.Code, w.Body)
	}

	// List after revoke — empty (only pending)
	req = httptest.NewRequest(http.MethodGet, "/invitations", nil)
	req = withOrgRoleContext(req, orgID, "admin")
	w = httptest.NewRecorder()
	ih.List(w, req)
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Data) != 0 {
		t.Errorf("list after revoke: expected 0, got %d", len(listResp.Data))
	}
}

func TestInvitation_AcceptFlow(t *testing.T) {
	ih, _ := newInvitationHandler(t)
	agentToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, agentToken)

	// Create invitation
	body, _ := json.Marshal(map[string]string{
		"email": "newtrust@example.com", "full_name": "New Trustee",
		"role": "trustee", "scheme_id": schemeID,
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req = withOrgRoleContext(req, orgID, "admin")
	w := httptest.NewRecorder()
	ih.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}

	// Read token directly from DB
	ctx := context.Background()
	pool := &database.Pool{Pool: testDB, Q: dbgen.New(testDB)}
	oid, err := uuid.Parse(orgID)
	if err != nil {
		t.Fatalf("uuid.Parse orgID: %v", err)
	}
	invs, err := pool.Q.ListInvitationsByOrg(ctx, oid)
	if err != nil || len(invs) == 0 {
		t.Fatalf("no invitations in DB: err=%v, count=%d", err, len(invs))
	}
	token := invs[0].Token

	// Verify endpoint
	req = httptest.NewRequest(http.MethodGet, "/invitations/"+token, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	ih.Verify(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", w.Code, w.Body)
	}
	var vResp struct{ Data invitation.VerifyResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&vResp)
	if vResp.Data.Email != "newtrust@example.com" {
		t.Errorf("verify: email=%q", vResp.Data.Email)
	}

	// Accept invitation
	acceptBody, _ := json.Marshal(map[string]string{"password": testPassword})
	req = httptest.NewRequest(http.MethodPost, "/invitations/"+token+"/accept", bytes.NewReader(acceptBody))
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	ih.Accept(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("accept: %d %s", w.Code, w.Body)
	}
	var authResp struct{ Data auth.AuthResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&authResp)
	if authResp.Data.AccessToken == "" {
		t.Fatal("accept: missing access token")
	}

	// Validate JWT — role must be trustee
	claims, err := auth.ValidateAccessToken(authResp.Data.AccessToken, testJWTSigningKey)
	if err != nil {
		t.Fatalf("accept: invalid token: %v", err)
	}
	if claims.Role != "trustee" {
		t.Errorf("accept: role=%q, want trustee", claims.Role)
	}

	// Verify after acceptance → 401
	req = httptest.NewRequest(http.MethodGet, "/invitations/"+token, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	ih.Verify(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("verify after accept: status=%d, want 401", w.Code)
	}
}
