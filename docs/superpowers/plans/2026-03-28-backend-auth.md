# Backend Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement full JWT auth for the Go backend — register (user+org), login, refresh, logout, and /me — replacing all stub handlers.

**Architecture:** Short-lived HS256 JWT (15 min) + rotating opaque refresh tokens stored in `refresh_tokens` table (7 days). The `Servicer` interface decouples handlers from the DB for unit testing. Context keys live in the `auth` package; `middleware/auth.go` imports `auth` to avoid an import cycle.

**Tech Stack:** `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, Chi router, pgx/v5, sqlc-generated queries, `database.WithTxQueries` for transactions.

---

### Task 1: Create branch and install dependencies

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum`

- [ ] **Step 1: Create feature branch**

```bash
git checkout -b feat/backend-auth
```

- [ ] **Step 2: Install JWT and crypto packages**

```bash
cd backend
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
```

Expected: `go.mod` now lists both packages under `require`.

- [ ] **Step 3: Verify the project still builds**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "chore(auth): add golang-jwt and bcrypt dependencies"
```

---

### Task 2: tokens.go — TDD

**Files:**
- Modify: `backend/internal/auth/tokens.go`
- Create: `backend/internal/auth/tokens_test.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/auth/tokens_test.go`:

```go
package auth

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-testing-only"

func TestGenerateAccessToken_ValidClaims(t *testing.T) {
	tok, err := GenerateAccessToken("user-123", "org-456", "admin", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := ValidateAccessToken(tok, testSecret)
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("sub = %q, want %q", claims.Subject, "user-123")
	}
	if claims.OrgID != "org-456" {
		t.Errorf("org_id = %q, want %q", claims.OrgID, "org-456")
	}
	if claims.Role != "admin" {
		t.Errorf("role = %q, want %q", claims.Role, "admin")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	tok, _ := GenerateAccessToken("user-1", "org-1", "admin", "secret-a", 15*time.Minute)
	_, err := ValidateAccessToken(tok, "secret-b")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	claims := Claims{
		OrgID: "org-1",
		Role:  "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to build expired token: %v", err)
	}
	_, err = ValidateAccessToken(tok, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_TamperedSignature(t *testing.T) {
	tok, _ := GenerateAccessToken("user-1", "org-1", "admin", testSecret, 15*time.Minute)
	tampered := tok[:len(tok)-4] + "xxxx"
	_, err := ValidateAccessToken(tampered, testSecret)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestGenerateRefreshToken_LengthAndEntropy(t *testing.T) {
	tok, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tok) != 64 {
		t.Errorf("token length = %d, want 64", len(tok))
	}
	if _, err := hex.DecodeString(tok); err != nil {
		t.Errorf("token is not valid hex: %v", err)
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	tok1, _ := GenerateRefreshToken()
	tok2, _ := GenerateRefreshToken()
	if tok1 == tok2 {
		t.Error("expected unique tokens, got identical values")
	}
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
cd backend
go test ./internal/auth/... -run TestGenerate -v 2>&1 | head -20
```

Expected: compile error — `Claims`, `GenerateAccessToken`, `ValidateAccessToken`, `GenerateRefreshToken` undefined.

- [ ] **Step 3: Implement tokens.go**

Replace the entire contents of `backend/internal/auth/tokens.go`:

```go
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is the type for auth-related context keys.
// Defined here (not in middleware) to avoid import cycles.
type ContextKey string

const (
	UserIDKey ContextKey = "user_id"
	OrgIDKey  ContextKey = "org_id"
	RoleKey   ContextKey = "role"
)

// Claims holds JWT payload fields beyond the registered set.
type Claims struct {
	OrgID string `json:"org_id"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed HS256 JWT for the given user.
func GenerateAccessToken(userID, orgID, role, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		OrgID: orgID,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken returns a 32-byte cryptographically random hex string.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ValidateAccessToken parses and validates a JWT, returning its claims.
func ValidateAccessToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd backend
go test ./internal/auth/... -v -run "TestGenerate|TestValidate"
```

Expected:
```
--- PASS: TestGenerateAccessToken_ValidClaims
--- PASS: TestValidateAccessToken_WrongSecret
--- PASS: TestValidateAccessToken_Expired
--- PASS: TestValidateAccessToken_TamperedSignature
--- PASS: TestGenerateRefreshToken_LengthAndEntropy
--- PASS: TestGenerateRefreshToken_Unique
PASS
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/auth/tokens.go backend/internal/auth/tokens_test.go
git commit -m "feat(auth): implement JWT + refresh token generation and validation"
```

---

### Task 3: service.go — types, errors, Servicer interface, updated struct

**Files:**
- Modify: `backend/internal/auth/service.go`

This task establishes the contract (interface + response types) that Task 4's handler tests depend on. No business logic yet.

- [ ] **Step 1: Replace service.go with scaffold**

```go
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/platform/database"
)

// Sentinel errors.
var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// Response types returned by the service.

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type OrgInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"` // seconds
	User         UserInfo `json:"user"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type MeResponse struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Org      OrgInfo `json:"org"`
	Role     string  `json:"role"`
}

// Servicer is the interface Handler depends on (enables test mocks).
type Servicer interface {
	Register(ctx context.Context, email, password, fullName, orgName string) (*AuthResponse, error)
	Login(ctx context.Context, email, password string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID, orgID string) (*MeResponse, error)
}

// Service implements Servicer.
type Service struct {
	db            *database.Pool
	cache         *redis.Client
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

// NewService constructs a Service. jwtSecret, jwtExpiry, and refreshExpiry
// come from config.Config.
func NewService(db *database.Pool, cache *redis.Client, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		cache:         cache,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend
go build ./internal/auth/...
```

Expected: compile error in `cmd/server/main.go` because `auth.NewService` signature changed — that's expected and will be fixed in Task 6.

```bash
go build ./internal/auth/... 2>&1 | grep -v main
```

Expected: no errors in the auth package itself.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/auth/service.go
git commit -m "feat(auth): add service types, errors, and Servicer interface"
```

---

### Task 4: handler.go — TDD with mock service

**Files:**
- Modify: `backend/internal/auth/handler.go`
- Create: `backend/internal/auth/handler_test.go`

- [ ] **Step 1: Write the failing handler tests**

Create `backend/internal/auth/handler_test.go`:

```go
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
	registerFn func(ctx context.Context, email, password, fullName, orgName string) (*AuthResponse, error)
	loginFn    func(ctx context.Context, email, password string) (*AuthResponse, error)
	refreshFn  func(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	logoutFn   func(ctx context.Context, refreshToken string) error
	meFn       func(ctx context.Context, userID, orgID string) (*MeResponse, error)
}

func (m *mockService) Register(ctx context.Context, email, password, fullName, orgName string) (*AuthResponse, error) {
	return m.registerFn(ctx, email, password, fullName, orgName)
}
func (m *mockService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	return m.loginFn(ctx, email, password)
}
func (m *mockService) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	return m.refreshFn(ctx, refreshToken)
}
func (m *mockService) Logout(ctx context.Context, refreshToken string) error {
	return m.logoutFn(ctx, refreshToken)
}
func (m *mockService) Me(ctx context.Context, userID, orgID string) (*MeResponse, error) {
	return m.meFn(ctx, userID, orgID)
}

// helpers

func body(t *testing.T, m map[string]string) *bytes.Reader {
	t.Helper()
	b, _ := json.Marshal(m)
	return bytes.NewReader(b)
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
		registerFn: func(_ context.Context, _, _, _, _ string) (*AuthResponse, error) {
			return nil, ErrEmailExists
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "a@b.com", "password": "pass", "full_name": "A B", "org_name": "Org",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestRegister_Success(t *testing.T) {
	svc := &mockService{
		registerFn: func(_ context.Context, _, _, _, _ string) (*AuthResponse, error) {
			return &AuthResponse{
				AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 900,
				User: UserInfo{ID: "u1", Email: "a@b.com", FullName: "A B"},
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", body(t, map[string]string{
		"email": "a@b.com", "password": "pass", "full_name": "A B", "org_name": "Org",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	var resp struct {
		Data AuthResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data.AccessToken == "" {
		t.Error("expected access_token in response")
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
			return &RefreshResponse{AccessToken: "new", RefreshToken: "new-rt", ExpiresIn: 900}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/refresh", body(t, map[string]string{"refresh_token": "old"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Refresh(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
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
		return errors.New("not found")
	}}
	req := httptest.NewRequest(http.MethodPost, "/logout", body(t, map[string]string{"refresh_token": "gone"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204 even on service error", w.Code)
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
	ctx := context.WithValue(req.Context(), UserIDKey, "u1")
	ctx = context.WithValue(ctx, OrgIDKey, "o1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	NewHandler(svc).Me(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data MeResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data.Role != "admin" {
		t.Errorf("role = %q, want admin", resp.Data.Role)
	}
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
cd backend
go test ./internal/auth/... 2>&1 | head -10
```

Expected: compile errors — `NewHandler` signature mismatch, handler methods not fully implemented.

- [ ] **Step 3: Implement handler.go**

Replace the entire contents of `backend/internal/auth/handler.go`:

```go
package auth

import (
	"encoding/json"
	"net/http"

	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	OrgName  string `json:"org_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.OrgName == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "email, password, full_name, and org_name are required")
		return
	}

	res, err := h.service.Register(r.Context(), req.Email, req.Password, req.FullName, req.OrgName)
	if err != nil {
		if err == ErrEmailExists {
			response.Error(w, http.StatusConflict, "CONFLICT", "email already registered")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "registration failed")
		return
	}
	response.JSON(w, http.StatusCreated, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "email and password are required")
		return
	}

	res, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == ErrInvalidCredentials {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "login failed")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "refresh_token is required")
		return
	}

	res, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if err == ErrInvalidToken {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "token refresh failed")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "refresh_token is required")
		return
	}
	// Idempotent — ignore service errors (token may already be revoked)
	_ = h.service.Logout(r.Context(), req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(UserIDKey).(string)
	orgID, _ := r.Context().Value(OrgIDKey).(string)
	if userID == "" || orgID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}

	res, err := h.service.Me(r.Context(), userID, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch user")
		return
	}
	response.JSON(w, http.StatusOK, res)
}
```

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd backend
go test ./internal/auth/... -v -run "TestRegister|TestLogin|TestRefresh|TestLogout|TestMe"
```

Expected:
```
--- PASS: TestRegister_BadJSON
--- PASS: TestRegister_MissingFields
--- PASS: TestRegister_DuplicateEmail
--- PASS: TestRegister_Success
--- PASS: TestLogin_BadJSON
--- PASS: TestLogin_InvalidCredentials
--- PASS: TestLogin_Success
--- PASS: TestRefresh_MissingToken
--- PASS: TestRefresh_InvalidToken
--- PASS: TestRefresh_Success
--- PASS: TestLogout_NoContent
--- PASS: TestLogout_Idempotent
--- PASS: TestMe_MissingContext
--- PASS: TestMe_Success
PASS
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/auth/handler.go backend/internal/auth/handler_test.go
git commit -m "feat(auth): implement auth handlers with TDD"
```

---

### Task 5: service.go — business logic

**Files:**
- Modify: `backend/internal/auth/service.go`

- [ ] **Step 1: Add business logic methods to service.go**

Append the following to `backend/internal/auth/service.go` (after the `NewService` function):

```go
import (
	// add these to the existing import block:
	// "github.com/google/uuid"
	// "github.com/jackc/pgx/v5"
	// "golang.org/x/crypto/bcrypt"
	// dbgen "github.com/stratahq/backend/db/gen"
	// "github.com/stratahq/backend/internal/platform/database"
)
```

Replace the full `service.go` file with:

```go
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/platform/database"
)

// Sentinel errors.
var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// Response types.

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type OrgInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	User         UserInfo `json:"user"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type MeResponse struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Org      OrgInfo `json:"org"`
	Role     string  `json:"role"`
}

// Servicer is the interface Handler depends on.
type Servicer interface {
	Register(ctx context.Context, email, password, fullName, orgName string) (*AuthResponse, error)
	Login(ctx context.Context, email, password string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID, orgID string) (*MeResponse, error)
}

// Service implements Servicer.
type Service struct {
	db            *database.Pool
	cache         *redis.Client
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

func NewService(db *database.Pool, cache *redis.Client, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		cache:         cache,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *Service) Register(ctx context.Context, email, password, fullName, orgName string) (*AuthResponse, error) {
	_, err := s.db.Q.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	var user dbgen.User
	var org dbgen.Org

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		var txErr error
		user, txErr = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			FullName:     fullName,
		})
		if txErr != nil {
			return txErr
		}
		org, txErr = q.CreateOrg(ctx, orgName)
		if txErr != nil {
			return txErr
		}
		_, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: user.ID,
			OrgID:  org.ID,
			Role:   "admin",
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, org.ID.String(), "admin")
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	user, err := s.db.Q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	memberships, err := s.db.Q.ListOrgMembershipsByUser(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return nil, ErrInvalidCredentials
	}

	m := memberships[0]
	return s.issueTokens(ctx, user, m.OrgID.String(), m.Role)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	rt, err := s.db.Q.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	memberships, err := s.db.Q.ListOrgMembershipsByUser(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return nil, ErrInvalidToken
	}
	m := memberships[0]

	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if txErr := q.RevokeRefreshToken(ctx, refreshToken); txErr != nil {
			return txErr
		}
		_, txErr := q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
			Token:     newRefreshToken,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(s.refreshExpiry),
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := GenerateAccessToken(user.ID.String(), m.OrgID.String(), m.Role, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.db.Q.RevokeRefreshToken(ctx, refreshToken)
}

func (s *Service) Me(ctx context.Context, userID, orgID string) (*MeResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	org, err := s.db.Q.GetOrg(ctx, oid)
	if err != nil {
		return nil, err
	}
	membership, err := s.db.Q.GetOrgMembershipByUser(ctx, dbgen.GetOrgMembershipByUserParams{
		UserID: uid,
		OrgID:  oid,
	})
	if err != nil {
		return nil, err
	}

	return &MeResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		Org:      OrgInfo{ID: org.ID.String(), Name: org.Name},
		Role:     membership.Role,
	}, nil
}

// issueTokens creates a token pair and persists the refresh token.
func (s *Service) issueTokens(ctx context.Context, user dbgen.User, orgID, role string) (*AuthResponse, error) {
	accessToken, err := GenerateAccessToken(user.ID.String(), orgID, role, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	_, err = s.db.Q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
	})
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
		User: UserInfo{
			ID:       user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}
```

- [ ] **Step 2: Verify the auth package compiles**

```bash
cd backend
go build ./internal/auth/...
```

Expected: no errors.

- [ ] **Step 3: Run the full unit test suite**

```bash
cd backend
go test ./internal/auth/... -v
```

Expected: all 14 tests pass (6 token tests + 14 handler tests — handler tests use mock, not DB).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/auth/service.go
git commit -m "feat(auth): implement service business logic (register, login, refresh, logout, me)"
```

---

### Task 6: Wire updated constructor + add /me to router

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/server/router.go`

- [ ] **Step 1: Update main.go — pass JWT config to NewService**

In `backend/cmd/server/main.go`, find:

```go
authService := auth.NewService(db, rdb)
```

Replace with:

```go
authService := auth.NewService(db, rdb, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)
```

- [ ] **Step 2: Add GET /auth/me to the protected group in router.go**

In `backend/internal/server/router.go`, find the protected group:

```go
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Mount("/schemes", h.Scheme.Routes())
```

Replace with:

```go
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Get("/auth/me", h.Auth.Me)
			r.Mount("/schemes", h.Scheme.Routes())
```

- [ ] **Step 3: Verify the full project builds**

```bash
cd backend
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/server/router.go
git commit -m "feat(auth): wire JWT config into service, add GET /auth/me to protected routes"
```

---

### Task 7: middleware/auth.go — real JWT validation

**Files:**
- Modify: `backend/internal/middleware/auth.go`

The existing middleware validates header format only. This task replaces the placeholder with real JWT validation. Because `middleware` will now import `auth`, we remove the `contextKey`/`UserIDKey` definitions from this file — they live in the `auth` package (added in Task 2).

- [ ] **Step 1: Replace middleware/auth.go**

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

// Auth validates the Bearer JWT and writes user_id, org_id, and role
// into the request context using keys from the auth package.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
				return
			}

			claims, err := auth.ValidateAccessToken(tokenStr, jwtSecret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}

			ctx := r.Context()
			ctx = contextWith(ctx, auth.UserIDKey, claims.Subject)
			ctx = contextWith(ctx, auth.OrgIDKey, claims.OrgID)
			ctx = contextWith(ctx, auth.RoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

Add the helper at the bottom of the same file (avoids importing `context` explicitly in the function signature):

```go
import "context"

func contextWith(ctx context.Context, key, val any) context.Context {
	return context.WithValue(ctx, key, val)
}
```

Actually, write the complete file as one block to avoid confusion:

```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

// Auth validates the Bearer JWT and injects user_id, org_id, and role
// into the request context. Context keys are defined in the auth package
// to avoid import cycles (handler reads them; middleware writes them).
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
				return
			}

			claims, err := auth.ValidateAccessToken(tokenStr, jwtSecret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), auth.UserIDKey, claims.Subject)
			ctx = context.WithValue(ctx, auth.OrgIDKey, claims.OrgID)
			ctx = context.WithValue(ctx, auth.RoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 2: Build the full project**

```bash
cd backend
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run all unit tests**

```bash
cd backend
go test ./internal/... -v -race
```

Expected: all tests pass including middleware tests (`TestLoggingMiddleware`, `TestRecoverMiddleware`).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/middleware/auth.go
git commit -m "feat(auth): wire real JWT validation into auth middleware"
```

---

### Task 8: Integration tests, final verification, and PR

**Files:**
- Create: `backend/tests/integration/auth_test.go`

- [ ] **Step 1: Write auth integration tests**

Create `backend/tests/integration/auth_test.go`:

```go
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
```

- [ ] **Step 2: Run unit tests (no DB needed)**

```bash
cd backend
go test ./internal/... -v -race
```

Expected: all pass.

- [ ] **Step 3: Run integration tests (requires running Postgres + Redis)**

Start services if not running:
```bash
docker compose up -d
```

Run migrations if needed:
```bash
make migrate
```

Run integration tests:
```bash
cd backend
go test ./tests/integration/... -v -race -tags=integration
```

Expected:
```
--- PASS: TestAuth_RegisterLoginRefreshLogout
--- PASS: TestAuth_WrongPassword
--- PASS: TestAuth_Me
--- PASS: TestHealthz_Integration
--- PASS: TestReadyz_Integration
PASS
```

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/auth_test.go
git commit -m "test(auth): add integration tests for full auth flow"
```

- [ ] **Step 5: Push branch and open PR**

```bash
git push -u origin feat/backend-auth
gh pr create \
  --title "feat(auth): implement backend JWT authentication" \
  --body "$(cat <<'EOF'
## Summary
- Implements register (user + org in one transaction), login, refresh (rotating tokens), logout, and GET /me
- Short-lived HS256 JWT (15 min) + opaque refresh tokens stored in \`refresh_tokens\` table (7 days, single-use rotation)
- \`Servicer\` interface decouples handlers from DB — 14 handler unit tests use a mock, 3 integration tests hit real Postgres
- Auth middleware now performs real JWT validation and injects \`user_id\`, \`org_id\`, \`role\` into request context

## Test plan
- [ ] \`go test ./internal/... -race\` passes (unit tests, no DB required)
- [ ] \`go test ./tests/integration/... -tags=integration -race\` passes with Docker Compose running
- [ ] \`POST /api/v1/auth/register\` → 201 with token pair
- [ ] \`POST /api/v1/auth/login\` with wrong password → 401
- [ ] \`POST /api/v1/auth/refresh\` rotates token; old token returns 401
- [ ] \`POST /api/v1/auth/logout\` → 204; subsequent refresh → 401
- [ ] \`GET /api/v1/auth/me\` with valid Bearer token → 200 with user + org + role
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- ✅ `POST /register` → user + org transaction, auto-login (Task 5)
- ✅ `POST /login` → bcrypt verify, token pair (Task 5)
- ✅ `POST /refresh` → rotation, new pair (Task 5)
- ✅ `POST /logout` → revoke, idempotent (Task 5)
- ✅ `GET /me` → user + org + role, protected (Tasks 4, 6)
- ✅ HS256 JWT, 15-min TTL, custom claims (Task 2)
- ✅ Opaque refresh token, 7-day TTL, server-side storage (Task 5)
- ✅ Middleware real JWT validation (Task 7)
- ✅ Import cycle avoided via context keys in auth package (Task 2)
- ✅ `golang-jwt/jwt/v5` + `bcrypt` (Task 1)
- ✅ Handler unit tests with mock (Task 4)
- ✅ Token unit tests (Task 2)
- ✅ Integration tests: register→login→refresh→logout + Me (Task 8)

**Type consistency:** `AuthResponse`, `RefreshResponse`, `MeResponse`, `UserInfo`, `OrgInfo` defined once in Task 3 service.go and referenced consistently in Tasks 4, 5, 8. `Claims.OrgID`/`Claims.Subject` used consistently in tokens.go and middleware. `UserIDKey`/`OrgIDKey` defined in tokens.go (Task 2) and used in handler.go (Task 4), middleware (Task 7), and integration tests (Task 8).
