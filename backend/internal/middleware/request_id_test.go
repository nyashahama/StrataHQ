package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDGeneratesAndPropagatesID(t *testing.T) {
	var seen string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if seen == "" {
		t.Fatal("request id in context is empty")
	}
	if got := w.Header().Get(RequestIDHeader); got != seen {
		t.Fatalf("response request id = %q, want %q", got, seen)
	}
}

func TestRequestIDUsesSafeIncomingID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != "req-client-123" {
			t.Fatalf("request id = %q, want req-client-123", got)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(RequestIDHeader, "req-client-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestRequestIDReplacesUnsafeIncomingID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := RequestIDFromContext(r.Context())
		if got == "bad\nid" {
			t.Fatal("unsafe request id was accepted")
		}
		if strings.ContainsAny(got, "\r\n\t ") {
			t.Fatalf("generated request id contains unsafe characters: %q", got)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(RequestIDHeader, "bad\nid")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestRequestIDFromContextReturnsEmptyWhenMissing(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Fatalf("request id = %q, want empty", got)
	}
}
