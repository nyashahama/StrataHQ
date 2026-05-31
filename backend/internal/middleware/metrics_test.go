package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRouteLabelUsesChiPatternWhenAvailable(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/schemes/123", nil)
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/api/v1/schemes/{schemeId}"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	if got := routeLabel(req); got != "/api/v1/schemes/{schemeId}" {
		t.Fatalf("route label = %q, want %q", got, "/api/v1/schemes/{schemeId}")
	}
}

func TestRouteLabelFallsBackToUnmatched(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz/abc/def", nil)

	if got := routeLabel(req); got != "unmatched" {
		t.Fatalf("route label = %q, want %q", got, "unmatched")
	}
}
