//go:build integration

package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/earlyaccess"
	"github.com/stratahq/backend/internal/notification"
)

func newEarlyAccessHandler(t *testing.T, adminEmail, adminSecret string) *earlyaccess.Handler {
	t.Helper()

	sender := &notification.NoopSender{}
	authSvc := auth.NewService(
		testPool,
		testRedis,
		sender,
		testJWTSigningKey,
		"http://localhost:3000",
		15*time.Minute,
		7*24*time.Hour,
	)
	svc := earlyaccess.NewService(
		testQ,
		authSvc,
		sender,
		"http://localhost:8080",
		"http://localhost:3000",
		adminEmail,
		adminSecret,
	)

	return earlyaccess.NewHandler(svc)
}

func createEarlyAccessRequest(t *testing.T) string {
	t.Helper()

	row, err := testQ.CreateEarlyAccessRequest(context.Background(), testCreateEarlyAccessRequestParams(t))
	if err != nil {
		t.Fatalf("CreateEarlyAccessRequest: %v", err)
	}

	return row.ID.String()
}

func testCreateEarlyAccessRequestParams(t *testing.T) dbgen.CreateEarlyAccessRequestParams {
	t.Helper()
	return dbgen.CreateEarlyAccessRequestParams{
		FullName:   "Early Access User",
		Email:      uniqueEmail(t),
		SchemeName: "Security Towers",
		UnitCount:  24,
	}
}

func signEarlyAccessAction(secret, id, action string, exp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s|%s|%d", id, action, exp)))
	return hex.EncodeToString(mac.Sum(nil))
}

func loadEarlyAccessStatus(t *testing.T, id string) string {
	t.Helper()

	uid, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("uuid.Parse: %v", err)
	}
	row, err := testQ.GetEarlyAccessRequest(context.Background(), uid)
	if err != nil {
		t.Fatalf("GetEarlyAccessRequest: %v", err)
	}
	return string(row.Status)
}

func TestEarlyAccess_ListRejectsTenantAdmin(t *testing.T) {
	h := newEarlyAccessHandler(t, "platform-admin@test.example.com", "platform-secret")
	accessToken, _ := setupAgent(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", w.Code, w.Body.String())
	}
}

func TestEarlyAccess_SignedApprovalGetDoesNotMutate(t *testing.T) {
	h := newEarlyAccessHandler(t, "platform-admin@test.example.com", "platform-secret")
	router := h.PublicRoutes()

	requestID := createEarlyAccessRequest(t)
	exp := time.Now().Add(15 * time.Minute).Unix()
	sig := signEarlyAccessAction("platform-secret", requestID, "approve", exp)

	req := httptest.NewRequest(http.MethodGet, "/"+requestID+"/approve?exp="+strconv.FormatInt(exp, 10)+"&sig="+sig, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", w.Code, w.Body.String())
	}
	if got := loadEarlyAccessStatus(t, requestID); got != "pending" {
		t.Fatalf("status after GET=%s, want pending", got)
	}
}

func TestEarlyAccess_SignedApprovalPostApprovesOnce(t *testing.T) {
	h := newEarlyAccessHandler(t, "platform-admin@test.example.com", "platform-secret")
	router := h.PublicRoutes()

	requestID := createEarlyAccessRequest(t)
	exp := time.Now().Add(15 * time.Minute).Unix()
	sig := signEarlyAccessAction("platform-secret", requestID, "approve", exp)
	url := "/" + requestID + "/approve?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + sig

	req := httptest.NewRequest(http.MethodPost, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first POST status=%d body=%s, want 200", w.Code, w.Body.String())
	}
	if got := loadEarlyAccessStatus(t, requestID); got != "approved" {
		t.Fatalf("status after POST=%s, want approved", got)
	}

	req = httptest.NewRequest(http.MethodPost, url, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("second POST status=%d body=%s, want 401", w.Code, w.Body.String())
	}
}

func TestEarlyAccess_SignedApprovalRejectsEmptyAdminSecret(t *testing.T) {
	h := newEarlyAccessHandler(t, "platform-admin@test.example.com", "")
	router := h.PublicRoutes()

	requestID := createEarlyAccessRequest(t)
	exp := time.Now().Add(15 * time.Minute).Unix()
	sig := signEarlyAccessAction("", requestID, "approve", exp)
	req := httptest.NewRequest(http.MethodPost, "/"+requestID+"/approve?exp="+strconv.FormatInt(exp, 10)+"&sig="+sig, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", w.Code, w.Body.String())
	}
	if got := loadEarlyAccessStatus(t, requestID); got != "pending" {
		t.Fatalf("status after empty-secret POST=%s, want pending", got)
	}
}
