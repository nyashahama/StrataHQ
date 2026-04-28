package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/stratahq/backend/internal/audit"
	"github.com/stratahq/backend/internal/auth"
)

type fakeAuditRecorder struct {
	events []audit.Event
}

func (f *fakeAuditRecorder) Record(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func TestAuditEvents_RecordsWriteRequests(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	handler := AuditEvents(recorder, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/org", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "audit-test")
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "user-123", "org-456", "admin"))

	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/api/v1/auth/org"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if len(recorder.events) != 1 {
		t.Fatalf("events recorded = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.ActorUserID != "user-123" {
		t.Fatalf("ActorUserID = %q, want %q", event.ActorUserID, "user-123")
	}
	if event.OrgID != "org-456" {
		t.Fatalf("OrgID = %q, want %q", event.OrgID, "org-456")
	}
	if event.Method != http.MethodPatch {
		t.Fatalf("Method = %q, want %q", event.Method, http.MethodPatch)
	}
	if event.Path != "/api/v1/auth/org" {
		t.Fatalf("Path = %q, want %q", event.Path, "/api/v1/auth/org")
	}
	if event.StatusCode != http.StatusNoContent {
		t.Fatalf("StatusCode = %d, want %d", event.StatusCode, http.StatusNoContent)
	}
	if event.IPAddress != "127.0.0.1" {
		t.Fatalf("IPAddress = %q, want %q", event.IPAddress, "127.0.0.1")
	}
	if event.UserAgent != "audit-test" {
		t.Fatalf("UserAgent = %q, want %q", event.UserAgent, "audit-test")
	}
}

func TestAuditEvents_SkipsReadRequests(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	handler := AuditEvents(recorder, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if len(recorder.events) != 0 {
		t.Fatalf("events recorded = %d, want 0", len(recorder.events))
	}
}

type discardWriter struct{}

func (w *discardWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type failingAuditRecorder struct{}

func (f failingAuditRecorder) Record(_ context.Context, _ audit.Event) error {
	return errors.New("audit store unavailable")
}

func TestAuditEvents_LogsRecorderFailureWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	handler := RequestID(AuditEvents(failingAuditRecorder{}, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/org", nil)
	req.Header.Set(RequestIDHeader, "req-audit-123")
	req.RemoteAddr = "127.0.0.1:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !bytes.Contains(buf.Bytes(), []byte(`"request_id":"req-audit-123"`)) {
		t.Fatalf("audit failure log missing request_id: %s", buf.String())
	}
}
