//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

func newAuthHandler(t *testing.T) *auth.Handler {
	t.Helper()
	pool := &database.Pool{Pool: testDB, Q: dbgen.New(testDB)}
	svc := auth.NewService(pool, testRedis, "integration-test-secret", 15*time.Minute, 7*24*time.Hour)
	return auth.NewHandler(svc)
}

func uniqueEmail(t *testing.T) string {
	safe := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	return fmt.Sprintf("%s@test.example.com", safe)
}

func TestAuth_RegisterLoginRefreshLogout(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// --- Register ---
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": "password123",
		"full_name": "Integration User", "org_name": "Integration Org",
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
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": "password123"})
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
		"full_name": "User", "org_name": "Org",
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
		"email": email, "password": "pass123",
		"full_name": "Me User", "org_name": "Me Org",
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

	// Parse claims from the access token to get userID and orgID
	claims, err := auth.ValidateAccessToken(regResp.Data.AccessToken, "integration-test-secret")
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}

	// Call Me with context values the middleware would normally inject
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	ctx := context.WithValue(req.Context(), auth.UserIDKey, claims.Subject)
	ctx = context.WithValue(ctx, auth.OrgIDKey, claims.OrgID)
	req = req.WithContext(ctx)
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
	if meResp.Data.Org.Name != "Me Org" {
		t.Errorf("me: org.name=%q, want 'Me Org'", meResp.Data.Org.Name)
	}
}
