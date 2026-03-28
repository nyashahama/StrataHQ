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
	h := health.New(testDB, &redisChecker{testRedis})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Healthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestReadyz_Integration(t *testing.T) {
	h := health.New(testDB, &redisChecker{testRedis})
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
