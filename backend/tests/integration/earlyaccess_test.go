//go:build integration

package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/earlyaccess"
	"github.com/stratahq/backend/internal/notification"
)

type fakeEarlyAccessAuthService struct {
	registerCalls int
	resetURLCalls int
}

func (f *fakeEarlyAccessAuthService) Register(_ context.Context, _, _, _ string) (*auth.AuthResponse, error) {
	f.registerCalls++
	return &auth.AuthResponse{}, nil
}

func (f *fakeEarlyAccessAuthService) IssuePasswordResetURL(_ context.Context, _, _ string) (string, error) {
	f.resetURLCalls++
	return "http://localhost:3000/auth/reset-password?token=test-token", nil
}

func (f *fakeEarlyAccessAuthService) Login(context.Context, string, string) (*auth.AuthResponse, error) {
	return nil, errors.New("unexpected Login call")
}

func (f *fakeEarlyAccessAuthService) Refresh(context.Context, string) (*auth.RefreshResponse, error) {
	return nil, errors.New("unexpected Refresh call")
}

func (f *fakeEarlyAccessAuthService) Logout(context.Context, string) error {
	return errors.New("unexpected Logout call")
}

func (f *fakeEarlyAccessAuthService) Me(context.Context, string, string) (*auth.MeResponse, error) {
	return nil, errors.New("unexpected Me call")
}

func (f *fakeEarlyAccessAuthService) Setup(context.Context, string, string, string, string, string, int32) (*auth.SetupResponse, error) {
	return nil, errors.New("unexpected Setup call")
}

func (f *fakeEarlyAccessAuthService) ForgotPassword(context.Context, string) error {
	return errors.New("unexpected ForgotPassword call")
}

func (f *fakeEarlyAccessAuthService) ResetPassword(context.Context, string, string) error {
	return errors.New("unexpected ResetPassword call")
}

func (f *fakeEarlyAccessAuthService) UpdateProfile(context.Context, string, string, string, string, *string) (*auth.MeResponse, error) {
	return nil, errors.New("unexpected UpdateProfile call")
}

func (f *fakeEarlyAccessAuthService) UpdateOrg(context.Context, string, string, *string, *string) (*auth.OrgInfo, error) {
	return nil, errors.New("unexpected UpdateOrg call")
}

func (f *fakeEarlyAccessAuthService) ChangePassword(context.Context, string, string, string) (*auth.RefreshResponse, error) {
	return nil, errors.New("unexpected ChangePassword call")
}

func (f *fakeEarlyAccessAuthService) ReissueSession(context.Context, string) (*auth.RefreshResponse, error) {
	return nil, errors.New("unexpected ReissueSession call")
}

func newEarlyAccessHandler(t *testing.T, adminEmail, adminSecret string) *earlyaccess.Handler {
	t.Helper()

	sender := &notification.NoopSender{}
	authSvc := auth.NewService(
		testPool,
		testRedis,
		sender,
		testJWTSigningKey,
		"http://localhost:3000",
		"http://localhost:3000",
		"stratahq-api",
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
	if w.Code != http.StatusConflict {
		t.Fatalf("second POST status=%d body=%s, want 409", w.Code, w.Body.String())
	}
}

func TestEarlyAccess_ProtectedReviewActionsOnlyTransitionPendingRequests(t *testing.T) {
	adminEmail := uniqueEmail(t)
	adminUser, err := testQ.CreateUser(t.Context(), dbgen.CreateUserParams{
		Email:        adminEmail,
		PasswordHash: "test-hash",
		FullName:     "Platform Admin",
	})
	if err != nil {
		t.Fatalf("create platform admin: %v", err)
	}

	authSvc := &fakeEarlyAccessAuthService{}
	sender := &notification.NoopSender{}
	svc := earlyaccess.NewService(
		testQ,
		authSvc,
		sender,
		"http://localhost:8080",
		"http://localhost:3000",
		adminEmail,
		"platform-secret",
	)
	h := earlyaccess.NewHandler(svc)

	requestID := createEarlyAccessRequest(t)
	req := httptest.NewRequest(http.MethodPost, "/"+requestID+"/approve", nil)
	req = withRouteParams(req, map[string]string{"id": requestID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), adminUser.ID.String(), uuid.NewString(), string(auth.RoleAdmin)))
	w := httptest.NewRecorder()

	h.Approve(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first approve status=%d body=%s, want 200", w.Code, w.Body.String())
	}
	if got := loadEarlyAccessStatus(t, requestID); got != "approved" {
		t.Fatalf("status after approve=%s, want approved", got)
	}
	if authSvc.registerCalls != 1 || authSvc.resetURLCalls != 1 || len(sender.InvitationsSent) != 1 {
		t.Fatalf("approval side effects register=%d resetURL=%d emails=%d, want 1 each", authSvc.registerCalls, authSvc.resetURLCalls, len(sender.InvitationsSent))
	}

	req = httptest.NewRequest(http.MethodPost, "/"+requestID+"/reject", nil)
	req = withRouteParams(req, map[string]string{"id": requestID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), adminUser.ID.String(), uuid.NewString(), string(auth.RoleAdmin)))
	w = httptest.NewRecorder()

	h.Reject(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("stale reject status=%d body=%s, want 409", w.Code, w.Body.String())
	}
	if got := loadEarlyAccessStatus(t, requestID); got != "approved" {
		t.Fatalf("status after stale reject=%s, want approved", got)
	}
	if authSvc.registerCalls != 1 || authSvc.resetURLCalls != 1 || len(sender.InvitationsSent) != 1 {
		t.Fatalf("stale review should not repeat approval side effects register=%d resetURL=%d emails=%d", authSvc.registerCalls, authSvc.resetURLCalls, len(sender.InvitationsSent))
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

func TestEarlyAccess_SubmitReturnsExistingPendingRequestOnDuplicateEmail(t *testing.T) {
	h := newEarlyAccessHandler(t, "platform-admin@test.example.com", "platform-secret")
	router := h.PublicRoutes()

	email := uniqueEmail(t)
	body := `{"full_name":"Person Example","email":"` + email + `","scheme_name":"Green View Estate","unit_count":12}`

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	first := httptest.NewRecorder()
	router.ServeHTTP(first, req)
	if first.Code != http.StatusCreated {
		t.Fatalf("first submit status=%d body=%s, want 201", first.Code, first.Body.String())
	}
	firstResp := decodeSuccess[earlyaccess.RequestResponse](t, first)

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	second := httptest.NewRecorder()
	router.ServeHTTP(second, req)
	if second.Code != http.StatusCreated {
		t.Fatalf("second submit status=%d body=%s, want 201", second.Code, second.Body.String())
	}
	secondResp := decodeSuccess[earlyaccess.RequestResponse](t, second)

	if firstResp.ID != secondResp.ID {
		t.Fatalf("expected duplicate submission to return same request id, got %q and %q", firstResp.ID, secondResp.ID)
	}
	if firstResp.Status != "pending" || secondResp.Status != "pending" {
		t.Fatalf("expected pending statuses, got %q and %q", firstResp.Status, secondResp.Status)
	}

	rows, err := testQ.ListEarlyAccessRequests(context.Background())
	if err != nil {
		t.Fatalf("ListEarlyAccessRequests: %v", err)
	}
	pendingCount := 0
	for _, row := range rows {
		if row.Email == email && row.Status == dbgen.EarlyAccessStatusPending {
			pendingCount++
		}
	}
	if pendingCount != 1 {
		t.Fatalf("expected one pending request for email %q, got %d", email, pendingCount)
	}
}
