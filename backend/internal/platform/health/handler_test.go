package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := New(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Healthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data["status"] != "ok" {
		t.Errorf("data.status = %q, want %q", resp.Data["status"], "ok")
	}
}

type fakeChecker struct {
	pingErr      error
	migrationErr error
}

func (f *fakeChecker) Ping(context.Context) error {
	return f.pingErr
}

func (f *fakeChecker) CheckMigrations(context.Context) error {
	return f.migrationErr
}

func TestReadyz_IntegrationLikeDependencyChecks(t *testing.T) {
	healthDB := &fakeChecker{}
	cache := &fakeChecker{}
	cache.pingErr = nil

	h := New(healthDB, cache, &fakeChecker{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	h.Readyz(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got := resp.Data["database"]; got != "ok" {
		t.Errorf("database check = %q, want ok", got)
	}
	if got := resp.Data["cache"]; got != "ok" {
		t.Errorf("cache check = %q, want ok", got)
	}
	if got := resp.Data["database_migrations"]; got != "ok" {
		t.Errorf("database_migrations check = %q, want ok", got)
	}
}

func TestReadyz_IntegrationLikeMigrationsFailure(t *testing.T) {
	db := &fakeChecker{}
	cache := &fakeChecker{}
	migrations := &fakeChecker{migrationErr: context.Canceled}

	h := New(db, cache, migrations)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	h.Readyz(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp struct {
		Err struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Err.Code != "NOT_READY" {
		t.Fatalf("error code = %q, want %q", resp.Err.Code, "NOT_READY")
	}
}
