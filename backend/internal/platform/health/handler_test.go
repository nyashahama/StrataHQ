package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := New(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Healthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data map[string]string `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data["status"] != "ok" {
		t.Errorf("data.status = %q, want %q", resp.Data["status"], "ok")
	}
}
