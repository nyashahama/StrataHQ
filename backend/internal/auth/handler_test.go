package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockService is a test double for Servicer.
type mockService struct {
	registerFn       func(ctx context.Context, email, password, fullName string) (*AuthResponse, error)
	loginFn          func(ctx context.Context, email, password string) (*AuthResponse, error)
	refreshFn        func(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	logoutFn         func(ctx context.Context, refreshToken string) error
	meFn             func(ctx context.Context, userID, orgID string) (*MeResponse, error)
	setupFn          func(ctx context.Context, orgID, orgName, contactEmail, schemeName, schemeAddress string, unitCount int32) (*SetupResponse, error)
	forgotFn         func(ctx context.Context, email string) error
	resetFn          func(ctx context.Context, token, password string) error
	updateProfileFn  func(ctx context.Context, userID, orgID, email, fullName string, phone *string) (*MeResponse, error)
	updateOrgFn      func(ctx context.Context, orgID, name string, contactEmail, contactPhone *string) (*OrgInfo, error)
	changePasswordFn func(ctx context.Context, userID, currentPassword, nextPassword string) error
}

func (m *mockService) Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error) {
	if m.registerFn == nil {
		return nil, nil
	}
	return m.registerFn(ctx, email, password, fullName)
}
func (m *mockService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	if m.loginFn == nil {
		return nil, nil
	}
	return m.loginFn(ctx, email, password)
}
func (m *mockService) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	if m.refreshFn == nil {
		return nil, nil
	}
	return m.refreshFn(ctx, refreshToken)
}
func (m *mockService) Logout(ctx context.Context, refreshToken string) error {
	if m.logoutFn == nil {
		return nil
	}
	return m.logoutFn(ctx, refreshToken)
}
func (m *mockService) Me(ctx context.Context, userID, orgID string) (*MeResponse, error) {
	if m.meFn == nil {
		return nil, nil
	}
	return m.meFn(ctx, userID, orgID)
}
func (m *mockService) Setup(ctx context.Context, orgID, orgName, contactEmail, schemeName, schemeAddress string, unitCount int32) (*SetupResponse, error) {
	if m.setupFn == nil {
		return nil, nil
	}
	return m.setupFn(ctx, orgID, orgName, contactEmail, schemeName, schemeAddress, unitCount)
}
func (m *mockService) ForgotPassword(ctx context.Context, email string) error {
	if m.forgotFn == nil {
		return nil
	}
	return m.forgotFn(ctx, email)
}
func (m *mockService) ResetPassword(ctx context.Context, token, password string) error {
	if m.resetFn == nil {
		return nil
	}
	return m.resetFn(ctx, token, password)
}
func (m *mockService) UpdateProfile(ctx context.Context, userID, orgID, email, fullName string, phone *string) (*MeResponse, error) {
	if m.updateProfileFn == nil {
		return nil, nil
	}
	return m.updateProfileFn(ctx, userID, orgID, email, fullName, phone)
}
func (m *mockService) UpdateOrg(ctx context.Context, orgID, name string, contactEmail, contactPhone *string) (*OrgInfo, error) {
	if m.updateOrgFn == nil {
		return nil, nil
	}
	return m.updateOrgFn(ctx, orgID, name, contactEmail, contactPhone)
}
func (m *mockService) ChangePassword(ctx context.Context, userID, currentPassword, nextPassword string) error {
	if m.changePasswordFn == nil {
		return nil
	}
	return m.changePasswordFn(ctx, userID, currentPassword, nextPassword)
}

func (m *mockService) IssuePasswordResetURL(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// helpers

func body(t *testing.T, m any) *bytes.Reader {
	t.Helper()
	b, _ := json.Marshal(m)
	return bytes.NewReader(b)
}

func strPtr(s string) *string {
	return &s
}

func TestRegister_UnknownFields(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"a@b.com","password":"Pass_1234","full_name":"A B","extra":"field"}`))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for unknown fields", w.Code)
	}
}

// --- Register ---

func TestRegister_BadJSON(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{"email": "a@b.com"}))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := &mockService{
		registerFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			return nil, ErrEmailExists
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "a@b.com", "password": "Pass_1234", "full_name": "A B",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestRegister_TrimsLeadingAndTrailingWhitespace(t *testing.T) {
	var capturedEmail, capturedFullName string
	svc := &mockService{
		registerFn: func(_ context.Context, email, _, fullName string) (*AuthResponse, error) {
			capturedEmail = email
			capturedFullName = fullName
			return &AuthResponse{
				AccessToken:  "access",
				RefreshToken: "refresh",
				ExpiresIn:    900,
				User:         UserInfo{ID: "u1", Email: email, FullName: fullName},
				Session:      MeResponse{ID: "u1", Email: email, FullName: fullName, Org: OrgInfo{ID: "o1", Name: "Org"}, Role: "admin"},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "  A@B.COM  ", "password": "Pass_1234", "full_name": "  New Name  ",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if capturedEmail != "a@b.com" {
		t.Fatalf("capturedEmail = %q, want %q", capturedEmail, "a@b.com")
	}
	if capturedFullName != "New Name" {
		t.Fatalf("capturedFullName = %q, want %q", capturedFullName, "New Name")
	}
}

func TestRegister_RejectsDisplayNameEmail(t *testing.T) {
	svcCalled := false
	svc := &mockService{
		registerFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			svcCalled = true
			return nil, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "User Example <user@example.com>", "password": "Pass_1234", "full_name": "User Example",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if svcCalled {
		t.Fatal("service should not be called for display-name email addresses")
	}
}

func TestRegister_RejectsWhitespaceOnlyRequiredFields(t *testing.T) {
	svcCalled := false
	svc := &mockService{
		registerFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			svcCalled = true
			return nil, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "   ", "password": "Pass_1234", "full_name": "   ",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if svcCalled {
		t.Fatalf("service should not be called with whitespace-only required fields")
	}
}

func TestRegister_Success(t *testing.T) {
	svc := &mockService{
		registerFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			return &AuthResponse{
				AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 900,
				User: UserInfo{ID: "u1", Email: "a@b.com", FullName: "A B"},
				Session: MeResponse{
					ID: "u1", Email: "a@b.com", FullName: "A B",
					Role: "admin", Org: OrgInfo{ID: "o1", Name: "Org"},
				},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "a@b.com", "password": "Pass_1234", "full_name": "A B",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	var resp struct {
		Data AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.AccessToken == "" {
		t.Error("expected access_token in response")
	}
	if resp.Data.Session.ID == "" {
		t.Error("expected session in response")
	}
}

func TestSetup_TrimsRequiredFieldsAndCallsService(t *testing.T) {
	var capturedOrgName, capturedContactEmail, capturedSchemeName, capturedSchemeAddress string
	var capturedUnitCount int32
	svc := &mockService{
		setupFn: func(_ context.Context, _, orgName, contactEmail, schemeName, schemeAddress string, unitCount int32) (*SetupResponse, error) {
			capturedOrgName = orgName
			capturedContactEmail = contactEmail
			capturedSchemeName = schemeName
			capturedSchemeAddress = schemeAddress
			capturedUnitCount = unitCount
			return &SetupResponse{
				Org: OrgInfo{
					ID:           "o1",
					Name:         orgName,
					ContactEmail: strPtr("admin@org.com"),
				},
				Scheme: struct {
					ID   string "json:\"id\""
					Name string "json:\"name\""
				}{ID: "s1", Name: "Scheme Name"},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/setup", body(t, map[string]interface{}{
		"org_name":       "  Test Org  ",
		"contact_email":  "  admin@org.com  ",
		"scheme_name":    "  Scheme Name  ",
		"scheme_address": "  10 Downing St  ",
		"unit_count":     12,
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()

	NewHandler(svc).Setup(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if capturedOrgName != "Test Org" {
		t.Fatalf("capturedOrgName = %q, want %q", capturedOrgName, "Test Org")
	}
	if capturedContactEmail != "admin@org.com" {
		t.Fatalf("capturedContactEmail = %q, want %q", capturedContactEmail, "admin@org.com")
	}
	if capturedSchemeName != "Scheme Name" {
		t.Fatalf("capturedSchemeName = %q, want %q", capturedSchemeName, "Scheme Name")
	}
	if capturedSchemeAddress != "10 Downing St" {
		t.Fatalf("capturedSchemeAddress = %q, want %q", capturedSchemeAddress, "10 Downing St")
	}
	if capturedUnitCount != 12 {
		t.Fatalf("capturedUnitCount = %d, want 12", capturedUnitCount)
	}
}

func TestSetup_RejectsWhitespaceOnlyRequiredFields(t *testing.T) {
	called := false
	svc := &mockService{
		setupFn: func(_ context.Context, _ string, _ string, _ string, _ string, _ string, _ int32) (*SetupResponse, error) {
			called = true
			return nil, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/setup", body(t, map[string]interface{}{
		"org_name":       "   ",
		"contact_email":  "admin@org.com",
		"scheme_name":    "   ",
		"scheme_address": "   ",
		"unit_count":     12,
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()
	NewHandler(svc).Setup(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if called {
		t.Fatal("service should not be called with whitespace-only required fields")
	}
}

// --- Login ---

func TestLogin_BadJSON(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("bad"))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestLogin_ReturnsSession(t *testing.T) {
	var capturedEmail string
	svc := &mockService{
		loginFn: func(_ context.Context, email, _ string) (*AuthResponse, error) {
			capturedEmail = email
			return &AuthResponse{
				AccessToken:  "access",
				RefreshToken: "refresh",
				ExpiresIn:    900,
				Session: MeResponse{
					ID:       "u1",
					Email:    "a@b.com",
					FullName: "A B",
					Role:     "admin",
					Org:      OrgInfo{ID: "o1", Name: "Org"},
				},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", body(t, map[string]string{
		"email": " A@B.COM ", "password": "pass",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Login(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Session.ID == "" {
		t.Error("expected session.id in login response")
	}
	if resp.Data.Session.Role != "admin" {
		t.Errorf("session.role = %q, want admin", resp.Data.Session.Role)
	}
	if resp.Data.Session.Org.ID == "" {
		t.Error("expected session.org in login response")
	}
	if capturedEmail != "a@b.com" {
		t.Fatalf("capturedEmail = %q, want %q", capturedEmail, "a@b.com")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := &mockService{
		loginFn: func(_ context.Context, _, _ string) (*AuthResponse, error) {
			return nil, ErrInvalidCredentials
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", body(t, map[string]string{
		"email": "a@b.com", "password": "wrong",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := &mockService{
		loginFn: func(_ context.Context, _, _ string) (*AuthResponse, error) {
			return &AuthResponse{
				AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 900,
				User: UserInfo{ID: "u1", Email: "a@b.com", FullName: "A B"},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", body(t, map[string]string{
		"email": "a@b.com", "password": "pass",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Login(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// --- Refresh ---

func TestRefresh_MissingToken(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/refresh", body(t, map[string]string{}))
	w := httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := &mockService{
		refreshFn: func(_ context.Context, _ string) (*RefreshResponse, error) {
			return nil, ErrInvalidToken
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/refresh", body(t, map[string]string{"refresh_token": "bad"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRefresh_Success(t *testing.T) {
	svc := &mockService{
		refreshFn: func(_ context.Context, _ string) (*RefreshResponse, error) {
			return &RefreshResponse{
				AccessToken:  "new",
				RefreshToken: "new-rt",
				ExpiresIn:    900,
				Session: MeResponse{
					ID: "u1", Email: "a@b.com", FullName: "A B",
					Role: "admin", Org: OrgInfo{ID: "o1", Name: "Org"},
				},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/refresh", body(t, map[string]string{"refresh_token": "old"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Refresh(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data RefreshResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Session.ID == "" {
		t.Error("expected session in refresh response")
	}
}

// --- Logout ---

func TestLogout_NoContent(t *testing.T) {
	svc := &mockService{logoutFn: func(_ context.Context, _ string) error { return nil }}
	req := httptest.NewRequest(http.MethodPost, "/logout", body(t, map[string]string{"refresh_token": "tok"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestLogout_Idempotent(t *testing.T) {
	svc := &mockService{logoutFn: func(_ context.Context, _ string) error {
		return ErrInvalidToken
	}}
	req := httptest.NewRequest(http.MethodPost, "/logout", body(t, map[string]string{"refresh_token": "gone"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204 even on service error", w.Code)
	}
}

func TestLogout_ReturnsServerErrorWhenRevocationFails(t *testing.T) {
	svc := &mockService{logoutFn: func(_ context.Context, _ string) error {
		return errors.New("db unavailable")
	}}
	req := httptest.NewRequest(http.MethodPost, "/logout", body(t, map[string]string{"refresh_token": "tok"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Logout(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// --- Me ---

func TestMe_MissingContext(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestMe_Success(t *testing.T) {
	svc := &mockService{
		meFn: func(_ context.Context, _, _ string) (*MeResponse, error) {
			return &MeResponse{
				ID: "u1", Email: "a@b.com", FullName: "A B",
				Org: OrgInfo{ID: "o1", Name: "Org"}, Role: "admin",
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()
	NewHandler(svc).Me(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data MeResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Role != "admin" {
		t.Errorf("role = %q, want admin", resp.Data.Role)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	svc := &mockService{
		updateProfileFn: func(_ context.Context, userID, orgID, email, fullName string, phone *string) (*MeResponse, error) {
			if userID != "u1" || orgID != "o1" {
				t.Fatalf("unexpected identity: user=%s org=%s", userID, orgID)
			}
			if email != "new@example.com" || fullName != "New Name" {
				t.Fatalf("unexpected payload: %s %s", email, fullName)
			}
			if phone == nil || *phone != "082 555 0101" {
				t.Fatalf("unexpected phone: %#v", phone)
			}
			return &MeResponse{
				ID:       userID,
				Email:    email,
				FullName: fullName,
				Phone:    phone,
				Org:      OrgInfo{ID: orgID, Name: "Org"},
				Role:     "resident",
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/profile", body(t, map[string]string{
		"email":     " New@Example.COM ",
		"full_name": " New Name ",
		"phone":     "082 555 0101",
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleResident)))
	w := httptest.NewRecorder()

	NewHandler(svc).UpdateProfile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestUpdateOrg_RejectsInvalidContactEmail(t *testing.T) {
	called := false
	svc := &mockService{
		updateOrgFn: func(context.Context, string, string, *string, *string) (*OrgInfo, error) {
			called = true
			return nil, nil
		},
	}
	raw := "not-an-email"
	req := httptest.NewRequest(http.MethodPatch, "/org", body(t, map[string]any{
		"name":          "Updated Org",
		"contact_email": raw,
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()

	NewHandler(svc).UpdateOrg(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if called {
		t.Fatal("service should not be called for invalid contact_email")
	}
}

func TestUpdateOrg_NormalizesContactEmail(t *testing.T) {
	var capturedContactEmail *string
	svc := &mockService{
		updateOrgFn: func(_ context.Context, orgID, name string, contactEmail, contactPhone *string) (*OrgInfo, error) {
			if orgID != "o1" || name != "Updated Org" {
				t.Fatalf("unexpected org update: orgID=%s name=%s", orgID, name)
			}
			capturedContactEmail = contactEmail
			return &OrgInfo{ID: orgID, Name: name, ContactEmail: contactEmail, ContactPhone: contactPhone}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPatch, "/org", body(t, map[string]string{
		"name":          " Updated Org ",
		"contact_email": " Admin@Example.COM ",
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()

	NewHandler(svc).UpdateOrg(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if capturedContactEmail == nil || *capturedContactEmail != "admin@example.com" {
		t.Fatalf("contactEmail = %#v, want admin@example.com", capturedContactEmail)
	}
}

func TestUpdateOrg_ForbiddenForNonAdmin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/org", body(t, map[string]string{
		"name": "Updated Org",
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleResident)))
	w := httptest.NewRecorder()

	NewHandler(&mockService{}).UpdateOrg(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestChangePassword_WrongPassword(t *testing.T) {
	svc := &mockService{
		changePasswordFn: func(_ context.Context, _, _, _ string) error {
			return ErrWrongPassword
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/change-password", body(t, map[string]string{
		"current_password": "Wrong_123",
		"new_password":     "NewSecret_1",
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()

	NewHandler(svc).ChangePassword(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "a@b.com", "password": "short", "full_name": "A B",
	}))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestResetPassword_WeakPassword(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/reset-password", body(t, map[string]string{
		"token": "tok", "password": "short",
	}))
	w := httptest.NewRecorder()
	h.ResetPassword(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestChangePassword_WeakNewPassword(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/change-password", body(t, map[string]string{
		"current_password": "OldPass_1", "new_password": "short",
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()
	h.ChangePassword(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
