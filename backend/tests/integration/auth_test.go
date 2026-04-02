//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/notification"
)

const (
	testJWTSigningKey = "for-integration-tests-only"
	testPassword      = "Tr0ub4dor&3-test-only"
)

func newAuthHandler(t *testing.T) *auth.Handler {
	t.Helper()
	sender := &notification.NoopSender{}
	svc := auth.NewService(testPool, testRedis, sender, testJWTSigningKey, "http://localhost:3000", 15*time.Minute, 7*24*time.Hour)
	return auth.NewHandler(svc)
}

func uniqueEmail(t *testing.T) string {
	safe := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x@test.example.com", safe, b)
}

func TestAuth_RegisterLoginRefreshLogout(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// --- Register ---
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Integration User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%s", w.Code, w.Body)
	}
	var regResp struct {
		Data auth.AuthResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&regResp)
	if regResp.Data.AccessToken == "" {
		t.Fatal("register: missing access_token")
	}
	if regResp.Data.RefreshToken == "" {
		t.Fatal("register: missing refresh_token")
	}

	// --- Duplicate register → 409 ---
	req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w = httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate register: status=%d, want 409", w.Code)
	}

	// --- Login ---
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": testPassword})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: status=%d body=%s", w.Code, w.Body)
	}
	var loginResp struct {
		Data auth.AuthResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&loginResp)
	originalRefreshToken := loginResp.Data.RefreshToken

	// --- Refresh ---
	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": originalRefreshToken})
	req = httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(refreshBody))
	w = httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: status=%d body=%s", w.Code, w.Body)
	}
	var refreshResp struct {
		Data auth.RefreshResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&refreshResp)
	newRefreshToken := refreshResp.Data.RefreshToken
	if newRefreshToken == originalRefreshToken {
		t.Error("refresh: token was not rotated")
	}

	// --- Old token is now invalid (rotation) ---
	req = httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(refreshBody))
	w = httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("reused old token: status=%d, want 401", w.Code)
	}

	// --- Logout ---
	logoutBody, _ := json.Marshal(map[string]string{"refresh_token": newRefreshToken})
	req = httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(logoutBody))
	w = httptest.NewRecorder()
	h.Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("logout: status=%d, want 204", w.Code)
	}

	// --- Refresh after logout must fail ---
	req = httptest.NewRequest(http.MethodPost, "/refresh",
		bytes.NewReader([]byte(fmt.Sprintf(`{"refresh_token":%q}`, newRefreshToken))))
	w = httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("post-logout refresh: status=%d, want 401", w.Code)
	}
}

func TestAuth_WrongPassword(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": "correct",
		"full_name": "User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register failed: %s", w.Body)
	}

	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": "wrong"})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: status=%d, want 401", w.Code)
	}
}

func TestAuth_Me(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword,
		"full_name": "Me User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register failed: %s", w.Body)
	}
	var regResp struct {
		Data auth.AuthResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&regResp)

	// Call Me with context values the middleware would normally inject
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req = withAuthContext(req, regResp.Data.AccessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Me(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me: status=%d body=%s", w.Code, w.Body)
	}

	var meResp struct {
		Data auth.MeResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&meResp)
	if meResp.Data.Email != email {
		t.Errorf("me: email=%q, want %q", meResp.Data.Email, email)
	}
	if meResp.Data.Role != "admin" {
		t.Errorf("me: role=%q, want admin", meResp.Data.Role)
	}
}

func TestAuth_SetupOnboarding(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// Register
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Setup User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%s", w.Code, w.Body)
	}
	var regResp struct {
		Data auth.AuthResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&regResp)
	accessToken := regResp.Data.AccessToken

	// /me before onboarding → wizard_complete=false, scheme_memberships=[]
	req = httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Me(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me pre-onboarding: status=%d body=%s", w.Code, w.Body)
	}
	var meResp struct {
		Data auth.MeResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&meResp)
	if meResp.Data.WizardComplete {
		t.Error("expected wizard_complete=false before onboarding")
	}
	if len(meResp.Data.SchemeMemberships) != 0 {
		t.Errorf("expected 0 scheme memberships before onboarding, got %d", len(meResp.Data.SchemeMemberships))
	}

	// POST /onboarding/setup
	setupBody, _ := json.Marshal(map[string]any{
		"org_name": "Sunset Heights", "contact_email": "admin@sunset.co.za",
		"scheme_name": "Sunset Heights Body Corporate", "scheme_address": "1 Kloof St, Cape Town",
		"unit_count": 12,
	})
	req = httptest.NewRequest(http.MethodPost, "/onboarding/setup", bytes.NewReader(setupBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status=%d body=%s", w.Code, w.Body)
	}
	var setupResp struct {
		Data auth.SetupResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&setupResp)
	if setupResp.Data.Org.Name != "Sunset Heights" {
		t.Errorf("setup: org name=%q, want Sunset Heights", setupResp.Data.Org.Name)
	}
	if setupResp.Data.Scheme.Name != "Sunset Heights Body Corporate" {
		t.Errorf("setup: scheme name=%q", setupResp.Data.Scheme.Name)
	}

	// /me after onboarding → wizard_complete=true, 1 scheme membership
	req = httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Me(w, req)
	json.NewDecoder(w.Body).Decode(&meResp)
	if !meResp.Data.WizardComplete {
		t.Error("expected wizard_complete=true after onboarding")
	}
	if len(meResp.Data.SchemeMemberships) != 1 {
		t.Errorf("expected 1 scheme membership, got %d", len(meResp.Data.SchemeMemberships))
	}

	// Non-admin role → 403
	req = httptest.NewRequest(http.MethodPost, "/onboarding/setup", bytes.NewReader(setupBody))
	req = withNonAdminContext(req)
	w = httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-admin setup: status=%d, want 403", w.Code)
	}
}

func TestAuth_ForgotResetPassword(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// Register user
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Reset User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}

	// ForgotPassword with known email → 200
	fpBody, _ := json.Marshal(map[string]string{"email": email})
	req = httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(fpBody))
	w = httptest.NewRecorder()
	h.ForgotPassword(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("forgot-password: status=%d, want 200", w.Code)
	}

	// ForgotPassword with unknown email → still 200 (no enumeration)
	fpBody, _ = json.Marshal(map[string]string{"email": "nobody@example.com"})
	req = httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(fpBody))
	w = httptest.NewRecorder()
	h.ForgotPassword(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("forgot-password unknown: status=%d, want 200", w.Code)
	}

	// Read token from Redis
	ctx := context.Background()
	keys, err := testRedis.Keys(ctx, "pwreset:*").Result()
	if err != nil || len(keys) == 0 {
		t.Fatal("no pwreset key in redis after forgot-password")
	}
	token := strings.TrimPrefix(keys[0], "pwreset:")

	// ResetPassword → 204
	rpBody, _ := json.Marshal(map[string]string{"token": token, "password": "NewP@ssw0rd!"})
	req = httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(rpBody))
	w = httptest.NewRecorder()
	h.ResetPassword(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("reset-password: status=%d body=%s, want 204", w.Code, w.Body)
	}

	// Login with new password succeeds
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": "NewP@ssw0rd!"})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("login after reset: status=%d, want 200", w.Code)
	}

	// Login with OLD password fails
	loginBody, _ = json.Marshal(map[string]string{"email": email, "password": testPassword})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("old password: status=%d, want 401", w.Code)
	}

	// Reuse token → 401
	req = httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(rpBody))
	w = httptest.NewRecorder()
	h.ResetPassword(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("reuse token: status=%d, want 401", w.Code)
	}
}

func TestAuth_ProfileOrgAndPasswordManagement(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	regBody, _ := json.Marshal(map[string]string{
		"email":     email,
		"password":  testPassword,
		"full_name": "Identity User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%s", w.Code, w.Body)
	}
	regResp := decodeSuccess[auth.AuthResponse](t, w)
	accessToken := regResp.AccessToken

	setupBody, _ := json.Marshal(map[string]any{
		"org_name":       "Identity Org",
		"contact_email":  "hello@identity.test",
		"scheme_name":    "Identity Scheme",
		"scheme_address": "1 Test Street",
		"unit_count":     4,
	})
	req = httptest.NewRequest(http.MethodPost, "/onboarding/setup", bytes.NewReader(setupBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status=%d body=%s", w.Code, w.Body)
	}

	profileBody, _ := json.Marshal(map[string]any{
		"email":     "updated+" + email,
		"full_name": "Updated Identity User",
		"phone":     "+27 82 555 0101",
	})
	req = httptest.NewRequest(http.MethodPatch, "/auth/profile", bytes.NewReader(profileBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.UpdateProfile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update profile: status=%d body=%s", w.Code, w.Body)
	}
	profileResp := decodeSuccess[auth.MeResponse](t, w)
	if profileResp.Email != "updated+"+email {
		t.Fatalf("updated email=%q", profileResp.Email)
	}
	if profileResp.Phone == nil || *profileResp.Phone != "+27 82 555 0101" {
		t.Fatalf("updated phone=%v", profileResp.Phone)
	}

	orgBody, _ := json.Marshal(map[string]any{
		"name":          "Updated Identity Org",
		"contact_email": "ops@identity.test",
		"contact_phone": "+27 21 555 0101",
	})
	req = httptest.NewRequest(http.MethodPatch, "/auth/org", bytes.NewReader(orgBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.UpdateOrg(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update org: status=%d body=%s", w.Code, w.Body)
	}
	orgResp := decodeSuccess[auth.OrgInfo](t, w)
	if orgResp.Name != "Updated Identity Org" {
		t.Fatalf("updated org name=%q", orgResp.Name)
	}
	if orgResp.ContactPhone == nil || *orgResp.ContactPhone != "+27 21 555 0101" {
		t.Fatalf("updated org phone=%v", orgResp.ContactPhone)
	}

	passwordBody, _ := json.Marshal(map[string]string{
		"current_password": testPassword,
		"new_password":     "N3wPassword!2026",
	})
	req = httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(passwordBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.ChangePassword(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("change password: status=%d body=%s", w.Code, w.Body)
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "updated+" + email,
		"password": "N3wPassword!2026",
	})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login with new password: status=%d body=%s", w.Code, w.Body)
	}
}
