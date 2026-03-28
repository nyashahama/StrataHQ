# Go Backend Structure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold the complete Go backend project structure with working infrastructure (config, server, middleware, Docker, CI, tests) and placeholder domain packages — ready for contributors to add business logic.

**Architecture:** Flat domain packages under `internal/`, shared platform infrastructure, sqlc for type-safe SQL, Chi router with layered middleware. Each domain follows handler/service/routes pattern.

**Tech Stack:** Go 1.24, Chi, PostgreSQL 17 (pgx/v5), sqlc, goose, Redis 7, Stripe Go SDK, Resend, OpenAI Go SDK (DeepSeek), slog, Prometheus, Docker, GitHub Actions, golangci-lint

---

### Task 1: Initialize Go Module & Project Skeleton

**Files:**
- Create: `backend/go.mod`
- Create: `backend/.gitignore`
- Create: `backend/.env.example`
- Create: `backend/cmd/server/main.go` (minimal — just prints "starting")

- [ ] **Step 1: Create the backend directory and initialize the Go module**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app
mkdir -p backend/cmd/server
cd backend
go mod init github.com/stratahq/backend
```

- [ ] **Step 2: Create .gitignore**

Create `backend/.gitignore`:

```gitignore
# Binary
bin/
server

# Environment
.env

# sqlc generated code
db/gen/

# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work
go.work.sum

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
```

- [ ] **Step 3: Create .env.example**

Create `backend/.env.example`:

```bash
# =============================================================================
# StrataHQ Backend Environment Variables
# Copy this file to .env and fill in the values
# =============================================================================

# Server
PORT=8080                                    # Render sets this automatically
ENV=development                              # development | staging | production

# Database (local Docker Compose default)
DATABASE_URL=postgres://stratahq:stratahq@localhost:5432/stratahq?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# Auth
JWT_SECRET=change-me-in-production           # min 32 characters in production
JWT_EXPIRY=15m                               # access token TTL
REFRESH_EXPIRY=168h                          # refresh token TTL (7 days)

# Stripe (use test keys for development)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Resend
RESEND_API_KEY=re_...

# AI (DeepSeek - OpenAI-compatible)
AI_BASE_URL=https://api.deepseek.com/v1
AI_API_KEY=sk-...
AI_MODEL=deepseek-chat

# CORS (comma-separated origins)
ALLOWED_ORIGINS=http://localhost:3000
```

- [ ] **Step 4: Create minimal main.go**

Create `backend/cmd/server/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("StrataHQ backend starting...")
	os.Exit(0)
}
```

- [ ] **Step 5: Verify the module compiles**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go build ./cmd/server/
```

Expected: no output (success), produces a `server` binary.

- [ ] **Step 6: Commit**

```bash
git add backend/
git commit -m "feat(backend): initialize Go module and project skeleton"
```

---

### Task 2: Configuration Package

**Files:**
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/config/config_test.go`

- [ ] **Step 1: Write the config test**

Create `backend/internal/config/config_test.go`:

```go
package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_AllFieldsSet(t *testing.T) {
	envs := map[string]string{
		"PORT":                   "9090",
		"ENV":                    "production",
		"DATABASE_URL":           "postgres://user:pass@localhost:5432/db",
		"REDIS_URL":              "redis://localhost:6379",
		"JWT_SECRET":             "test-secret-that-is-long-enough-32ch",
		"JWT_EXPIRY":             "30m",
		"REFRESH_EXPIRY":         "48h",
		"STRIPE_SECRET_KEY":      "sk_test_123",
		"STRIPE_WEBHOOK_SECRET":  "whsec_123",
		"RESEND_API_KEY":         "re_123",
		"AI_BASE_URL":            "https://api.deepseek.com/v1",
		"AI_API_KEY":             "sk-ai-123",
		"AI_MODEL":               "deepseek-chat",
		"ALLOWED_ORIGINS":        "http://localhost:3000,https://app.stratahq.com",
	}

	for k, v := range envs {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want %q", cfg.Env, "production")
	}
	if cfg.DatabaseURL != envs["DATABASE_URL"] {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, envs["DATABASE_URL"])
	}
	if cfg.JWTExpiry != 30*time.Minute {
		t.Errorf("JWTExpiry = %v, want %v", cfg.JWTExpiry, 30*time.Minute)
	}
	if cfg.RefreshExpiry != 48*time.Hour {
		t.Errorf("RefreshExpiry = %v, want %v", cfg.RefreshExpiry, 48*time.Hour)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins length = %d, want 2", len(cfg.AllowedOrigins))
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Set only required fields
	required := map[string]string{
		"DATABASE_URL":           "postgres://user:pass@localhost:5432/db",
		"REDIS_URL":              "redis://localhost:6379",
		"JWT_SECRET":             "test-secret-that-is-long-enough-32ch",
		"STRIPE_SECRET_KEY":      "sk_test_123",
		"STRIPE_WEBHOOK_SECRET":  "whsec_123",
		"RESEND_API_KEY":         "re_123",
		"AI_BASE_URL":            "https://api.deepseek.com/v1",
		"AI_API_KEY":             "sk-ai-123",
		"AI_MODEL":               "deepseek-chat",
	}

	for k, v := range required {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// Clear optional fields to test defaults
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("JWT_EXPIRY")
	os.Unsetenv("REFRESH_EXPIRY")
	os.Unsetenv("ALLOWED_ORIGINS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
	if cfg.Env != "development" {
		t.Errorf("Env = %q, want default %q", cfg.Env, "development")
	}
	if cfg.JWTExpiry != 15*time.Minute {
		t.Errorf("JWTExpiry = %v, want default %v", cfg.JWTExpiry, 15*time.Minute)
	}
	if cfg.RefreshExpiry != 168*time.Hour {
		t.Errorf("RefreshExpiry = %v, want default %v", cfg.RefreshExpiry, 168*time.Hour)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// Clear all env vars
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET", "STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "RESEND_API_KEY", "AI_BASE_URL", "AI_API_KEY", "AI_MODEL"} {
		os.Unsetenv(key)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing required fields, got nil")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go test ./internal/config/ -v
```

Expected: FAIL — `Load` function not defined.

- [ ] **Step 3: Implement the config package**

Create `backend/internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	// Server
	Port string
	Env  string // "development", "staging", "production"

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Auth
	JWTSecret     string
	JWTExpiry     time.Duration
	RefreshExpiry time.Duration

	// Stripe
	StripeSecretKey     string
	StripeWebhookSecret string

	// Resend
	ResendAPIKey string

	// AI
	AIBaseURL string
	AIAPIKey  string
	AIModel   string

	// CORS
	AllowedOrigins []string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		Env:           getEnv("ENV", "development"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		ResendAPIKey:  os.Getenv("RESEND_API_KEY"),
		AIBaseURL:     os.Getenv("AI_BASE_URL"),
		AIAPIKey:      os.Getenv("AI_API_KEY"),
		AIModel:       os.Getenv("AI_MODEL"),
	}

	// Parse durations with defaults
	var err error
	cfg.JWTExpiry, err = parseDuration("JWT_EXPIRY", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.RefreshExpiry, err = parseDuration("REFRESH_EXPIRY", 168*time.Hour)
	if err != nil {
		return nil, err
	}

	// Parse CORS origins
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins != "" {
		cfg.AllowedOrigins = strings.Split(origins, ",")
		for i := range cfg.AllowedOrigins {
			cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":          c.DatabaseURL,
		"REDIS_URL":             c.RedisURL,
		"JWT_SECRET":            c.JWTSecret,
		"STRIPE_SECRET_KEY":     c.StripeSecretKey,
		"STRIPE_WEBHOOK_SECRET": c.StripeWebhookSecret,
		"RESEND_API_KEY":        c.ResendAPIKey,
		"AI_BASE_URL":           c.AIBaseURL,
		"AI_API_KEY":            c.AIAPIKey,
		"AI_MODEL":              c.AIModel,
	}

	var missing []string
	for name, val := range required {
		if val == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go test ./internal/config/ -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/config/
git commit -m "feat(backend): add configuration package with env var loading and validation"
```

---

### Task 3: Platform — Database, Redis, Response Helpers

**Files:**
- Create: `backend/internal/platform/database/database.go`
- Create: `backend/internal/platform/database/tx.go`
- Create: `backend/internal/platform/cache/redis.go`
- Create: `backend/internal/platform/response/json.go`
- Create: `backend/internal/platform/response/json_test.go`

- [ ] **Step 1: Write the JSON response helper test**

Create `backend/internal/platform/response/json_test.go`:

```go
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"name": "test"}

	JSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected data to be non-nil")
	}
}

func TestJSONList(t *testing.T) {
	w := httptest.NewRecorder()
	items := []string{"a", "b", "c"}
	meta := Meta{Page: 1, PerPage: 20, Total: 3}

	JSONList(w, http.StatusOK, items, meta)

	var resp SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Meta == nil {
		t.Error("expected meta to be non-nil")
	}
	if resp.Meta.Total != 3 {
		t.Errorf("meta.total = %d, want 3", resp.Meta.Total)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()

	Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid email")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Err.Code != "VALIDATION_ERROR" {
		t.Errorf("error.code = %q, want %q", resp.Err.Code, "VALIDATION_ERROR")
	}
	if resp.Err.Message != "invalid email" {
		t.Errorf("error.message = %q, want %q", resp.Err.Message, "invalid email")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go test ./internal/platform/response/ -v
```

Expected: FAIL — types and functions not defined.

- [ ] **Step 3: Implement response helpers**

Create `backend/internal/platform/response/json.go`:

```go
package response

import (
	"encoding/json"
	"net/http"
)

type SuccessResponse struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type ErrorResponse struct {
	Err ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(SuccessResponse{Data: data})
}

func JSONList(w http.ResponseWriter, status int, data any, meta Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(SuccessResponse{Data: data, Meta: &meta})
}

func Error(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Err: ErrorBody{Code: code, Message: message},
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/platform/response/ -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Implement database package**

Create `backend/internal/platform/database/database.go`:

```go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
```

- [ ] **Step 6: Implement transaction helper**

Create `backend/internal/platform/database/tx.go`:

```go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
```

- [ ] **Step 7: Implement Redis cache package**

Create `backend/internal/platform/cache/redis.go`:

```go
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

func HealthCheck(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}
```

- [ ] **Step 8: Fetch dependencies and verify compilation**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go get github.com/jackc/pgx/v5
go get github.com/redis/go-redis/v9
go mod tidy
go build ./...
```

Expected: compiles without errors.

- [ ] **Step 9: Commit**

```bash
git add backend/internal/platform/ backend/go.mod backend/go.sum
git commit -m "feat(backend): add platform packages — database, redis, response helpers"
```

---

### Task 4: Health Check Handler

**Files:**
- Create: `backend/internal/platform/health/handler.go`
- Create: `backend/internal/platform/health/handler_test.go`

- [ ] **Step 1: Write the health check test**

Create `backend/internal/platform/health/handler_test.go`:

```go
package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := New(nil, nil) // nil deps — healthz doesn't check them
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Healthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/platform/health/ -v
```

Expected: FAIL — `New` not defined.

- [ ] **Step 3: Implement health handler**

Create `backend/internal/platform/health/handler.go`:

```go
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/stratahq/backend/internal/platform/response"
)

type Checker interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db    Checker
	cache Checker
}

func New(db Checker, cache Checker) *Handler {
	return &Handler{db: db, cache: cache}
}

// Healthz is a liveness probe — always returns 200 if the process is running.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz is a readiness probe — checks database and cache connectivity.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := map[string]string{}
	healthy := true

	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["database"] = err.Error()
			healthy = false
		} else {
			checks["database"] = "ok"
		}
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			checks["cache"] = err.Error()
			healthy = false
		} else {
			checks["cache"] = "ok"
		}
	}

	if !healthy {
		response.Error(w, http.StatusServiceUnavailable, "NOT_READY", "one or more dependencies are unhealthy")
		return
	}

	response.JSON(w, http.StatusOK, checks)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/platform/health/ -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/health/
git commit -m "feat(backend): add health check handlers (healthz + readyz)"
```

---

### Task 5: Middleware Stack

**Files:**
- Create: `backend/internal/middleware/recover.go`
- Create: `backend/internal/middleware/logging.go`
- Create: `backend/internal/middleware/cors.go`
- Create: `backend/internal/middleware/ratelimit.go`
- Create: `backend/internal/middleware/metrics.go`
- Create: `backend/internal/middleware/auth.go`
- Create: `backend/internal/middleware/logging_test.go`
- Create: `backend/internal/middleware/recover_test.go`

- [ ] **Step 1: Write recover middleware test**

Create `backend/internal/middleware/recover_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecover_NoPanic(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRecover_WithPanic(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
```

- [ ] **Step 2: Write logging middleware test**

Create `backend/internal/middleware/logging_test.go`:

```go
package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if buf.Len() == 0 {
		t.Error("expected log output, got empty")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/middleware/ -v
```

Expected: FAIL — functions not defined.

- [ ] **Step 4: Implement recover middleware**

Create `backend/internal/middleware/recover.go`:

```go
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/stratahq/backend/internal/platform/response"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
					"method", r.Method,
					"path", r.URL.Path,
				)
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 5: Implement logging middleware**

Create `backend/internal/middleware/logging.go`:

```go
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}
```

- [ ] **Step 6: Implement CORS middleware**

Create `backend/internal/middleware/cors.go`:

```go
package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[strings.TrimSpace(o)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 7: Implement rate limit middleware**

Create `backend/internal/middleware/ratelimit.go`:

```go
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stratahq/backend/internal/platform/response"
)

func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := r.RemoteAddr
			key := fmt.Sprintf("ratelimit:%s", ip)
			ctx := context.Background()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// If Redis is down, allow the request
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				rdb.Expire(ctx, key, window)
			}

			if count > int64(limit) {
				response.Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 8: Implement metrics middleware**

Create `backend/internal/middleware/metrics.go`:

```go
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}
```

- [ ] **Step 9: Implement auth middleware (placeholder — validates JWT structure)**

Create `backend/internal/middleware/auth.go`:

```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/stratahq/backend/internal/platform/response"
)

type contextKey string

const UserIDKey contextKey = "user_id"

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

			token := parts[1]
			if token == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
				return
			}

			// TODO: JWT validation will be implemented when auth domain is built.
			// For now, this middleware validates the header format only.
			ctx := context.WithValue(r.Context(), UserIDKey, "placeholder")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 10: Fetch dependencies and run tests**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
go mod tidy
go test ./internal/middleware/ -v
```

Expected: all middleware tests PASS.

- [ ] **Step 11: Commit**

```bash
git add backend/internal/middleware/ backend/go.mod backend/go.sum
git commit -m "feat(backend): add middleware stack — recover, logging, CORS, rate limit, metrics, auth"
```

---

### Task 6: Server & Router Wiring

**Files:**
- Create: `backend/internal/server/server.go`
- Create: `backend/internal/server/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Implement the router**

Create `backend/internal/server/router.go`:

```go
package server

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/middleware"
	"github.com/stratahq/backend/internal/platform/health"
)

func NewRouter(cfg *config.Config, logger *slog.Logger, healthHandler *health.Handler, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.Recover)
	r.Use(middleware.Metrics)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(rdb, 100, 1*time.Minute))

	// Health & metrics (outside /api/v1, no auth)
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			// Auth routes will be mounted here
			// Stripe webhook routes will be mounted here
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			// Domain routes will be mounted here:
			// r.Mount("/schemes", scheme.Routes())
			// r.Mount("/levies", levy.Routes())
			// r.Mount("/maintenance", maintenance.Routes())
			// r.Mount("/billing", billing.Routes())
		})
	})

	return r
}
```

- [ ] **Step 2: Implement the server with graceful shutdown**

Create `backend/internal/server/server.go`:

```go
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func New(handler http.Handler, port string, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Channel to listen for server errors
	serverErr := make(chan error, 1)

	go func() {
		s.logger.Info("server starting", "addr", s.httpServer.Addr)
		serverErr <- s.httpServer.ListenAndServe()
	}()

	// Block until we receive a signal or server error
	select {
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		s.logger.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.httpServer.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}
```

- [ ] **Step 3: Wire everything in main.go**

Replace `backend/cmd/server/main.go` with:

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/platform/cache"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/platform/health"
	"github.com/stratahq/backend/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Database
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// Redis
	rdb, err := cache.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("redis connected")

	// Health
	healthHandler := health.New(db, rdb)

	// Router & Server
	router := server.NewRouter(cfg, logger, healthHandler, rdb)
	srv := server.New(router, cfg.Port, logger)

	logger.Info("starting server", "port", cfg.Port, "env", cfg.Env)
	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
```

- [ ] **Step 4: Fetch Chi dependency and verify compilation**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go get github.com/go-chi/chi/v5
go mod tidy
go build ./cmd/server/
```

Expected: compiles without errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/server/ backend/cmd/server/main.go backend/go.mod backend/go.sum
git commit -m "feat(backend): add server, router, and main.go wiring with graceful shutdown"
```

---

### Task 7: Domain Scaffolds (auth, scheme, levy, maintenance, billing, notification, ai)

**Files:**
- Create: `backend/internal/auth/handler.go`
- Create: `backend/internal/auth/service.go`
- Create: `backend/internal/auth/tokens.go`
- Create: `backend/internal/auth/routes.go`
- Create: `backend/internal/scheme/handler.go`
- Create: `backend/internal/scheme/service.go`
- Create: `backend/internal/scheme/routes.go`
- Create: `backend/internal/levy/handler.go`
- Create: `backend/internal/levy/service.go`
- Create: `backend/internal/levy/routes.go`
- Create: `backend/internal/maintenance/handler.go`
- Create: `backend/internal/maintenance/service.go`
- Create: `backend/internal/maintenance/routes.go`
- Create: `backend/internal/billing/handler.go`
- Create: `backend/internal/billing/webhook.go`
- Create: `backend/internal/billing/service.go`
- Create: `backend/internal/billing/routes.go`
- Create: `backend/internal/notification/email.go`
- Create: `backend/internal/notification/templates.go`
- Create: `backend/internal/ai/client.go`
- Create: `backend/internal/ai/config.go`

Each domain follows the same pattern. I'll show the full code for `auth` (most complex) and `scheme` (standard pattern), then the remaining domains follow the identical structure.

- [ ] **Step 1: Create auth domain**

Create `backend/internal/auth/handler.go`:

```go
package auth

import (
	"net/http"

	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "register endpoint"})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "login endpoint"})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "refresh endpoint"})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "logout endpoint"})
}
```

Create `backend/internal/auth/service.go`:

```go
package auth

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

func NewService(db *pgxpool.Pool, cache *redis.Client) *Service {
	return &Service{db: db, cache: cache}
}
```

Create `backend/internal/auth/tokens.go`:

```go
package auth

// JWT token creation and validation will be implemented here.
// This file is scaffolded as part of the auth domain structure.
```

Create `backend/internal/auth/routes.go`:

```go
package auth

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)

	return r
}
```

- [ ] **Step 2: Create scheme domain**

Create `backend/internal/scheme/handler.go`:

```go
package scheme

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "list schemes"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "get scheme", "id": id})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusCreated, map[string]string{"message": "create scheme"})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "update scheme", "id": id})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "delete scheme", "id": id})
}
```

Create `backend/internal/scheme/service.go`:

```go
package scheme

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}
```

Create `backend/internal/scheme/routes.go`:

```go
package scheme

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}
```

- [ ] **Step 3: Create levy domain**

Create `backend/internal/levy/handler.go`:

```go
package levy

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "list levies"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "get levy", "id": id})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusCreated, map[string]string{"message": "create levy"})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "update levy", "id": id})
}
```

Create `backend/internal/levy/service.go`:

```go
package levy

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}
```

Create `backend/internal/levy/routes.go`:

```go
package levy

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)

	return r
}
```

- [ ] **Step 4: Create maintenance domain**

Create `backend/internal/maintenance/handler.go`:

```go
package maintenance

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "list maintenance requests"})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "get maintenance request", "id": id})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusCreated, map[string]string{"message": "create maintenance request"})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.JSON(w, http.StatusOK, map[string]string{"message": "update maintenance request", "id": id})
}
```

Create `backend/internal/maintenance/service.go`:

```go
package maintenance

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}
```

Create `backend/internal/maintenance/routes.go`:

```go
package maintenance

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)

	return r
}
```

- [ ] **Step 5: Create billing domain**

Create `backend/internal/billing/handler.go`:

```go
package billing

import (
	"net/http"

	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "create checkout session"})
}

func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "get subscription"})
}
```

Create `backend/internal/billing/webhook.go`:

```go
package billing

import (
	"net/http"

	"github.com/stratahq/backend/internal/platform/response"
)

func (h *Handler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "stripe webhook received"})
}
```

Create `backend/internal/billing/service.go`:

```go
package billing

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}
```

Create `backend/internal/billing/routes.go`:

```go
package billing

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/checkout", h.CreateCheckoutSession)
	r.Get("/subscription", h.GetSubscription)

	return r
}

func (h *Handler) WebhookRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/stripe", h.HandleStripeWebhook)

	return r
}
```

- [ ] **Step 6: Create notification package**

Create `backend/internal/notification/email.go`:

```go
package notification

type EmailClient struct {
	apiKey string
}

func NewEmailClient(apiKey string) *EmailClient {
	return &EmailClient{apiKey: apiKey}
}
```

Create `backend/internal/notification/templates.go`:

```go
package notification

// Email templates will be defined here.
// Each template is a function that returns a subject and HTML body.
```

- [ ] **Step 7: Create AI package**

Create `backend/internal/ai/client.go`:

```go
package ai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	client *openai.Client
	model  string
}

func NewClient(cfg Config) *Client {
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
	)

	return &Client{
		client: &client,
		model:  cfg.Model,
	}
}

func (c *Client) Chat(ctx context.Context, message string) (string, error) {
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(message),
		},
	})
	if err != nil {
		return "", fmt.Errorf("chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}
```

Create `backend/internal/ai/config.go`:

```go
package ai

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}
```

- [ ] **Step 8: Fetch dependencies and verify compilation**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go get github.com/openai/openai-go
go mod tidy
go build ./...
```

Expected: compiles without errors.

- [ ] **Step 9: Commit**

```bash
git add backend/internal/auth/ backend/internal/scheme/ backend/internal/levy/ backend/internal/maintenance/ backend/internal/billing/ backend/internal/notification/ backend/internal/ai/ backend/go.mod backend/go.sum
git commit -m "feat(backend): scaffold domain packages — auth, scheme, levy, maintenance, billing, notification, ai"
```

---

### Task 8: Wire Domain Routes into Router

**Files:**
- Modify: `backend/internal/server/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Update router to mount all domain routes**

Replace the contents of `backend/internal/server/router.go` with:

```go
package server

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/maintenance"
	"github.com/stratahq/backend/internal/middleware"
	"github.com/stratahq/backend/internal/platform/health"
	"github.com/stratahq/backend/internal/scheme"
)

type Handlers struct {
	Health      *health.Handler
	Auth        *auth.Handler
	Scheme      *scheme.Handler
	Levy        *levy.Handler
	Maintenance *maintenance.Handler
	Billing     *billing.Handler
}

func NewRouter(cfg *config.Config, logger *slog.Logger, rdb *redis.Client, h Handlers) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.Recover)
	r.Use(middleware.Metrics)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(rdb, 100, 1*time.Minute))

	// Health & metrics (outside /api/v1, no auth)
	r.Get("/healthz", h.Health.Healthz)
	r.Get("/readyz", h.Health.Readyz)
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Mount("/auth", h.Auth.Routes())
			r.Mount("/billing/webhooks", h.Billing.WebhookRoutes())
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Mount("/schemes", h.Scheme.Routes())
			r.Mount("/levies", h.Levy.Routes())
			r.Mount("/maintenance", h.Maintenance.Routes())
			r.Mount("/billing", h.Billing.Routes())
		})
	})

	return r
}
```

- [ ] **Step 2: Update main.go to wire domain services and handlers**

Replace `backend/cmd/server/main.go` with:

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/billing"
	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/maintenance"
	"github.com/stratahq/backend/internal/platform/cache"
	"github.com/stratahq/backend/internal/platform/database"
	"github.com/stratahq/backend/internal/platform/health"
	"github.com/stratahq/backend/internal/scheme"
	"github.com/stratahq/backend/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Database
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// Redis
	rdb, err := cache.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("redis connected")

	// Services
	authService := auth.NewService(db, rdb)
	schemeService := scheme.NewService(db)
	levyService := levy.NewService(db)
	maintenanceService := maintenance.NewService(db)
	billingService := billing.NewService(db)

	// Handlers
	handlers := server.Handlers{
		Health:      health.New(db, rdb),
		Auth:        auth.NewHandler(authService),
		Scheme:      scheme.NewHandler(schemeService),
		Levy:        levy.NewHandler(levyService),
		Maintenance: maintenance.NewHandler(maintenanceService),
		Billing:     billing.NewHandler(billingService),
	}

	// Router & Server
	router := server.NewRouter(cfg, logger, rdb, handlers)
	srv := server.New(router, cfg.Port, logger)

	logger.Info("starting server", "port", cfg.Port, "env", cfg.Env)
	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go build ./cmd/server/
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/server/ backend/cmd/server/main.go
git commit -m "feat(backend): wire all domain routes into router and main.go"
```

---

### Task 9: Database — sqlc Config, Placeholder Migration, Queries

**Files:**
- Create: `backend/db/sqlc.yaml`
- Create: `backend/db/migrations/00001_init.sql`
- Create: `backend/db/queries/auth.sql`
- Create: `backend/db/queries/scheme.sql`
- Create: `backend/db/queries/levy.sql`
- Create: `backend/db/queries/maintenance.sql`

- [ ] **Step 1: Create sqlc config**

Create `backend/db/sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "migrations/"
    gen:
      go:
        package: "dbgen"
        out: "gen"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
```

- [ ] **Step 2: Create placeholder init migration**

Create `backend/db/migrations/00001_init.sql`:

```sql
-- +goose Up
-- Placeholder init migration.
-- Domain-specific tables will be added as the project is implemented.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS schema_migrations_lock (
    id INTEGER PRIMARY KEY DEFAULT 1,
    locked BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down

DROP TABLE IF EXISTS schema_migrations_lock;
```

- [ ] **Step 3: Create placeholder query files**

Create `backend/db/queries/auth.sql`:

```sql
-- name: Placeholder :exec
-- Placeholder query — sqlc requires at least one query per file.
-- Replace with real queries when implementing the auth domain.
SELECT 1;
```

Create `backend/db/queries/scheme.sql`:

```sql
-- name: Placeholder :exec
-- Placeholder query — sqlc requires at least one query per file.
-- Replace with real queries when implementing the scheme domain.
SELECT 1;
```

Create `backend/db/queries/levy.sql`:

```sql
-- name: Placeholder :exec
-- Placeholder query — sqlc requires at least one query per file.
-- Replace with real queries when implementing the levy domain.
SELECT 1;
```

Create `backend/db/queries/maintenance.sql`:

```sql
-- name: Placeholder :exec
-- Placeholder query — sqlc requires at least one query per file.
-- Replace with real queries when implementing the maintenance domain.
SELECT 1;
```

- [ ] **Step 4: Install sqlc and verify generation**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
cd db && sqlc generate && cd ..
```

Expected: generates files in `db/gen/`.

- [ ] **Step 5: Commit (don't include db/gen/)**

```bash
git add backend/db/sqlc.yaml backend/db/migrations/ backend/db/queries/
git commit -m "feat(backend): add sqlc config, placeholder migration, and query files"
```

---

### Task 10: Docker Setup

**Files:**
- Create: `backend/Dockerfile`
- Create: `backend/docker-compose.yml`

- [ ] **Step 1: Create Dockerfile**

Create `backend/Dockerfile`:

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server/

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/server .
COPY --from=builder /app/db/migrations ./db/migrations

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:${PORT:-8080}/healthz || exit 1

ENTRYPOINT ["./server"]
```

- [ ] **Step 2: Create docker-compose.yml**

Create `backend/docker-compose.yml`:

```yaml
services:
  backend:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    env_file:
      - .env
    environment:
      - DATABASE_URL=postgres://stratahq:stratahq@postgres:5432/stratahq?sslmode=disable
      - REDIS_URL=redis://redis:6379

  postgres:
    image: postgres:17-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: stratahq
      POSTGRES_PASSWORD: stratahq
      POSTGRES_DB: stratahq
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U stratahq"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

- [ ] **Step 3: Verify Docker builds**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
docker build -t stratahq-backend .
```

Expected: image builds successfully.

- [ ] **Step 4: Commit**

```bash
git add backend/Dockerfile backend/docker-compose.yml
git commit -m "feat(backend): add Dockerfile and docker-compose for local dev"
```

---

### Task 11: Makefile & Scripts

**Files:**
- Create: `backend/Makefile`
- Create: `backend/scripts/migrate.sh`
- Create: `backend/scripts/generate.sh`
- Create: `backend/scripts/seed.sh`

- [ ] **Step 1: Create Makefile**

Create `backend/Makefile`:

```makefile
.PHONY: run build test test-integration lint generate migrate-up migrate-down migrate-create docker-up docker-down seed clean

# ============================================================================
# Development
# ============================================================================

run:
	go run ./cmd/server/

build:
	go build -ldflags="-s -w" -o bin/server ./cmd/server/

clean:
	rm -rf bin/

# ============================================================================
# Testing
# ============================================================================

test:
	go test ./internal/... -v -race

test-integration:
	go test ./tests/integration/... -v -race -tags=integration

test-all: test test-integration

# ============================================================================
# Code Quality
# ============================================================================

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w .

# ============================================================================
# Code Generation
# ============================================================================

generate:
	cd db && sqlc generate

# ============================================================================
# Database
# ============================================================================

migrate-up:
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-create:
	goose -dir db/migrations create $(name) sql

migrate-status:
	goose -dir db/migrations postgres "$(DATABASE_URL)" status

seed:
	@echo "Running seed script..."
	@bash scripts/seed.sh

# ============================================================================
# Docker
# ============================================================================

docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

docker-build:
	docker compose build backend

docker-all:
	docker compose up --build
```

- [ ] **Step 2: Create migration script**

Create `backend/scripts/migrate.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

MIGRATIONS_DIR="db/migrations"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

case "${1:-up}" in
    up)
        goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" up
        ;;
    down)
        goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" down
        ;;
    status)
        goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" status
        ;;
    *)
        echo "Usage: $0 {up|down|status}"
        exit 1
        ;;
esac
```

- [ ] **Step 3: Create generate script**

Create `backend/scripts/generate.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "Generating sqlc code..."
cd db && sqlc generate
echo "Done. Generated files in db/gen/"
```

- [ ] **Step 4: Create seed script**

Create `backend/scripts/seed.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

echo "Seeding database..."
# Add seed SQL commands here as the schema is built out.
# Example:
# psql "$DATABASE_URL" -f db/seed.sql
echo "No seed data configured yet. Add seed SQL to this script."
```

- [ ] **Step 5: Make scripts executable**

```bash
chmod +x backend/scripts/*.sh
```

- [ ] **Step 6: Commit**

```bash
git add backend/Makefile backend/scripts/
git commit -m "feat(backend): add Makefile and development scripts"
```

---

### Task 12: Linter Config & CI Pipeline

**Files:**
- Create: `backend/.golangci.yml`
- Create: `backend/.github/workflows/ci.yml`

- [ ] **Step 1: Create golangci-lint config**

Create `backend/.golangci.yml`:

```yaml
run:
  timeout: 5m
  go: "1.24"

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - bodyclose
    - nilerr
    - exportloopref

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    enable-all: true

issues:
  exclude-dirs:
    - db/gen
```

- [ ] **Step 2: Create CI workflow**

Create `backend/.github/workflows/ci.yml`:

```yaml
name: Backend CI

on:
  push:
    branches: [main]
    paths: ["backend/**"]
  pull_request:
    branches: [main]
    paths: ["backend/**"]

defaults:
  run:
    working-directory: backend

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          working-directory: backend

  test:
    name: Test
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_USER: stratahq
          POSTGRES_PASSWORD: stratahq
          POSTGRES_DB: stratahq_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U stratahq"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Install tools
        run: |
          go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
          go install github.com/pressly/goose/v3/cmd/goose@latest

      - name: Generate sqlc
        run: cd db && sqlc generate

      - name: Run migrations
        run: goose -dir db/migrations postgres "postgres://stratahq:stratahq@localhost:5432/stratahq_test?sslmode=disable" up

      - name: Unit tests
        run: go test ./internal/... -v -race

      - name: Integration tests
        run: go test ./tests/integration/... -v -race -tags=integration
        env:
          DATABASE_URL: postgres://stratahq:stratahq@localhost:5432/stratahq_test?sslmode=disable
          REDIS_URL: redis://localhost:6379

  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Build binary
        run: go build -ldflags="-s -w" -o bin/server ./cmd/server/

      - name: Build Docker image
        run: docker build -t stratahq-backend .
```

- [ ] **Step 3: Run lint locally to verify config**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
golangci-lint run ./...
```

Expected: passes (or reports issues to fix).

- [ ] **Step 4: Commit**

```bash
git add backend/.golangci.yml backend/.github/
git commit -m "feat(backend): add golangci-lint config and GitHub Actions CI pipeline"
```

---

### Task 13: Integration Test Scaffold

**Files:**
- Create: `backend/tests/integration/testhelpers_test.go`
- Create: `backend/tests/integration/health_test.go`

- [ ] **Step 1: Create test helpers**

Create `backend/tests/integration/testhelpers_test.go`:

```go
//go:build integration

package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	testDB    *pgxpool.Pool
	testRedis *redis.Client
	testLog   *slog.Logger
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	testLog = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://stratahq:stratahq@localhost:5432/stratahq?sslmode=disable"
	}

	var err error
	testDB, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		testLog.Error("failed to connect to test database", "error", err)
		os.Exit(1)
	}
	defer testDB.Close()

	// Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		testLog.Error("failed to parse redis URL", "error", err)
		os.Exit(1)
	}
	testRedis = redis.NewClient(opts)
	defer testRedis.Close()

	os.Exit(m.Run())
}
```

- [ ] **Step 2: Create health integration test**

Create `backend/tests/integration/health_test.go`:

```go
//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/platform/health"
)

func TestHealthz_Integration(t *testing.T) {
	h := health.New(testDB, testRedis)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Healthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestReadyz_Integration(t *testing.T) {
	h := health.New(testDB, testRedis)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	h.Readyz(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be an object")
	}

	if data["database"] != "ok" {
		t.Errorf("database = %v, want ok", data["database"])
	}
	if data["cache"] != "ok" {
		t.Errorf("cache = %v, want ok", data["cache"])
	}
}
```

- [ ] **Step 3: Create fixtures directory**

```bash
mkdir -p backend/tests/fixtures
echo '# Test fixtures go here' > backend/tests/fixtures/.gitkeep
```

- [ ] **Step 4: Run unit tests to make sure nothing is broken**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go test ./internal/... -v -race
```

Expected: all unit tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/tests/
git commit -m "feat(backend): add integration test scaffold with health check tests"
```

---

### Task 14: README & CONTRIBUTING

**Files:**
- Create: `backend/README.md`
- Create: `backend/CONTRIBUTING.md`

- [ ] **Step 1: Create README**

Create `backend/README.md`:

```markdown
# StrataHQ Backend

Go backend for [StrataHQ](https://stratahq.com) — property management software for South African sectional title schemes.

## Tech Stack

| Tool | Purpose |
|------|---------|
| [Go 1.24](https://go.dev/) | Language |
| [Chi](https://github.com/go-chi/chi) | HTTP router |
| [PostgreSQL 17](https://www.postgresql.org/) | Primary database |
| [pgx/v5](https://github.com/jackc/pgx) | Postgres driver |
| [sqlc](https://sqlc.dev/) | Type-safe SQL codegen |
| [goose](https://github.com/pressly/goose) | Database migrations |
| [Redis 7](https://redis.io/) | Caching, rate limiting, sessions |
| [Stripe](https://stripe.com/) | Payment processing |
| [Resend](https://resend.com/) | Transactional email |
| [DeepSeek](https://deepseek.com/) | AI features (OpenAI-compatible) |
| [Prometheus](https://prometheus.io/) | Metrics |
| [Docker](https://www.docker.com/) | Containerization |

## Quick Start

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)
- [goose](https://github.com/pressly/goose#install)
- [golangci-lint](https://golangci-lint.run/welcome/install/)

### Setup

```bash
# Clone the repo
git clone <repo-url>
cd stratahq-app/backend

# Copy environment config
cp .env.example .env

# Start Postgres + Redis
make docker-up

# Run database migrations
make migrate-up

# Generate sqlc code
make generate

# Start the server
make run
```

The server starts on `http://localhost:8080`. Verify with:

```bash
curl http://localhost:8080/healthz
# {"data":{"status":"ok"}}
```

## Project Structure

```
backend/
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── config/          # Environment variable loading & validation
│   ├── server/          # HTTP server setup, router, graceful shutdown
│   ├── middleware/       # Request middleware (auth, logging, CORS, metrics, etc.)
│   ├── auth/            # Authentication domain (JWT, login, register)
│   ├── scheme/          # Scheme management domain
│   ├── levy/            # Levy & payments domain
│   ├── maintenance/     # Maintenance requests domain
│   ├── billing/         # Stripe billing domain
│   ├── notification/    # Email notifications (Resend)
│   ├── ai/              # AI features (DeepSeek via OpenAI SDK)
│   └── platform/        # Shared infrastructure (database, cache, health, response)
├── db/
│   ├── migrations/      # Goose SQL migrations
│   ├── queries/         # sqlc query files
│   └── gen/             # Generated sqlc code (gitignored)
├── tests/
│   ├── integration/     # Integration tests (require Docker services)
│   └── fixtures/        # Test data
├── scripts/             # Development scripts
├── Dockerfile           # Multi-stage production build
├── docker-compose.yml   # Local development services
└── Makefile             # Development commands
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make run` | Run the server locally |
| `make build` | Build binary to `bin/server` |
| `make test` | Run unit tests |
| `make test-integration` | Run integration tests (requires Docker services) |
| `make test-all` | Run all tests |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make generate` | Generate sqlc code |
| `make migrate-up` | Run pending migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-create name=<name>` | Create a new migration |
| `make migrate-status` | Show migration status |
| `make docker-up` | Start Postgres + Redis containers |
| `make docker-down` | Stop containers |
| `make docker-all` | Build and start all services |
| `make seed` | Seed the development database |

## Adding a New Domain

1. Create a new package under `internal/`:
   ```
   internal/mydomain/
   ├── handler.go    # HTTP handlers
   ├── service.go    # Business logic
   └── routes.go     # Chi route definitions
   ```

2. Add sqlc queries in `db/queries/mydomain.sql`

3. Run `make generate` to regenerate the Go code

4. Wire the handler into `internal/server/router.go`

5. Wire the service into `cmd/server/main.go`

6. Add tests next to your code (`*_test.go`) and in `tests/integration/`

## API

All API routes are prefixed with `/api/v1`.

### Response Format

```json
// Success
{ "data": { ... }, "meta": { "page": 1, "per_page": 20, "total": 42 } }

// Error
{ "error": { "code": "VALIDATION_ERROR", "message": "..." } }
```

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/healthz` | No | Liveness probe |
| `GET` | `/readyz` | No | Readiness probe (checks DB + Redis) |
| `GET` | `/metrics` | No | Prometheus metrics |
| `POST` | `/api/v1/auth/register` | No | Register a new user |
| `POST` | `/api/v1/auth/login` | No | Login |
| `POST` | `/api/v1/auth/refresh` | No | Refresh access token |
| `POST` | `/api/v1/auth/logout` | No | Logout |
| `GET` | `/api/v1/schemes` | Yes | List schemes |
| `POST` | `/api/v1/schemes` | Yes | Create a scheme |
| `GET` | `/api/v1/schemes/:id` | Yes | Get a scheme |
| `PUT` | `/api/v1/schemes/:id` | Yes | Update a scheme |
| `DELETE` | `/api/v1/schemes/:id` | Yes | Delete a scheme |
| `GET` | `/api/v1/levies` | Yes | List levies |
| `POST` | `/api/v1/levies` | Yes | Create a levy |
| `GET` | `/api/v1/levies/:id` | Yes | Get a levy |
| `PUT` | `/api/v1/levies/:id` | Yes | Update a levy |
| `GET` | `/api/v1/maintenance` | Yes | List maintenance requests |
| `POST` | `/api/v1/maintenance` | Yes | Create a maintenance request |
| `GET` | `/api/v1/maintenance/:id` | Yes | Get a maintenance request |
| `PUT` | `/api/v1/maintenance/:id` | Yes | Update a maintenance request |
| `POST` | `/api/v1/billing/checkout` | Yes | Create Stripe checkout session |
| `GET` | `/api/v1/billing/subscription` | Yes | Get subscription status |
| `POST` | `/api/v1/billing/webhooks/stripe` | No | Stripe webhook |

## License

Proprietary. See LICENSE for details.
```

- [ ] **Step 2: Create CONTRIBUTING.md**

Create `backend/CONTRIBUTING.md`:

```markdown
# Contributing to StrataHQ Backend

Thank you for contributing! This guide will help you get started.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch: `git checkout -b feat/my-feature`
4. Follow the [Quick Start](README.md#quick-start) to set up your environment

## Development Workflow

### Branch Naming

- `feat/description` — new features
- `fix/description` — bug fixes
- `docs/description` — documentation
- `refactor/description` — code refactoring
- `test/description` — test additions or fixes

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(auth): add password reset endpoint
fix(levy): correct payment amount calculation
docs(readme): add deployment instructions
test(scheme): add integration tests for CRUD
refactor(middleware): simplify rate limit logic
```

### Code Style

- Run `make fmt` before committing
- Run `make lint` to check for issues
- CI enforces `golangci-lint` — your PR won't merge with lint errors

### Testing

- **Unit tests** go next to the code: `internal/mydomain/service_test.go`
- **Integration tests** go in `tests/integration/`
- Use table-driven tests:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"valid input", "hello", "HELLO"},
        {"empty input", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

- Run unit tests: `make test`
- Run integration tests: `make docker-up && make test-integration`
- All tests must pass before submitting a PR

### Adding a New Domain

See the [Adding a New Domain](README.md#adding-a-new-domain) section in the README.

## Pull Request Process

1. Ensure all tests pass (`make test-all`)
2. Ensure lint passes (`make lint`)
3. Update documentation if needed
4. Fill out the PR template
5. Request a review

### PR Template

```markdown
## What

Brief description of what this PR does.

## Why

Why is this change needed?

## How to Test

Steps to verify the change:
1. ...
2. ...
3. ...

## Checklist

- [ ] Tests added/updated
- [ ] Lint passes (`make lint`)
- [ ] Documentation updated (if applicable)
```

## Questions?

Open an issue or reach out to the maintainers.
```

- [ ] **Step 3: Commit**

```bash
git add backend/README.md backend/CONTRIBUTING.md
git commit -m "docs(backend): add README and CONTRIBUTING guide"
```

---

### Task 15: Final Verification

**Files:** None — this is a verification-only task.

- [ ] **Step 1: Verify the full project compiles**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go build ./...
```

Expected: no errors.

- [ ] **Step 2: Run all unit tests**

```bash
make test
```

Expected: all tests PASS.

- [ ] **Step 3: Run linter**

```bash
make lint
```

Expected: no errors.

- [ ] **Step 4: Verify Docker builds**

```bash
docker build -t stratahq-backend .
```

Expected: image builds successfully.

- [ ] **Step 5: Verify the directory structure matches the spec**

```bash
find . -type f | grep -v node_modules | grep -v '.git/' | grep -v 'db/gen/' | sort
```

Review output against the spec's directory structure.

- [ ] **Step 6: Final commit if any fixes were needed**

```bash
git status
# If there are changes:
git add -A
git commit -m "fix(backend): address issues found during final verification"
```
