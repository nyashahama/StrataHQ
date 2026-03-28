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
