package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPerEndpointRateLimitUsesDistinctPrefixes(t *testing.T) {
	testIP := "192.0.2.1"

	loginMw := PerEndpointRateLimit(nil, "login", 5, 0)
	refreshMw := PerEndpointRateLimit(nil, "refresh", 5, 0)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrappedLogin := loginMw(handler)
	wrappedRefresh := refreshMw(handler)

	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = testIP + ":12345"

	rec := httptest.NewRecorder()

	wrappedLogin.ServeHTTP(rec, req)
	wrappedRefresh.ServeHTTP(rec, req)

	wantLoginKey := fmt.Sprintf("ratelimit:login:%s", testIP)
	wantRefreshKey := fmt.Sprintf("ratelimit:refresh:%s", testIP)

	if wantLoginKey == wantRefreshKey {
		t.Fatalf("login (key=%q) and refresh (key=%q) must use distinct keys", wantLoginKey, wantRefreshKey)
	}
	if wantLoginKey != "ratelimit:login:192.0.2.1" {
		t.Fatalf("login key incorrect: got %q, want %q", wantLoginKey, "ratelimit:login:192.0.2.1")
	}
	if wantRefreshKey != "ratelimit:refresh:192.0.2.1" {
		t.Fatalf("refresh key incorrect: got %q, want %q", wantRefreshKey, "ratelimit:refresh:192.0.2.1")
	}
}

func TestRouterAppliesDistinctRateLimitPrefixesForLoginAndRefresh(t *testing.T) {
	testIP := "10.0.0.1"

	loginMw := PerEndpointRateLimit(nil, "auth-login", 5, 0)
	refreshMw := PerEndpointRateLimit(nil, "auth-refresh", 30, 0)

	loginHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	refreshHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrappedLogin := loginMw(loginHandler)
	wrappedRefresh := refreshMw(refreshHandler)

	loginReq := httptest.NewRequest("POST", "/auth/login", nil)
	loginReq.RemoteAddr = testIP + ":12345"
	loginRec := httptest.NewRecorder()

	refreshReq := httptest.NewRequest("POST", "/auth/refresh", nil)
	refreshReq.RemoteAddr = testIP + ":12345"
	refreshRec := httptest.NewRecorder()

	wrappedLogin.ServeHTTP(loginRec, loginReq)
	wrappedRefresh.ServeHTTP(refreshRec, refreshReq)

	wantLoginKey := fmt.Sprintf("ratelimit:auth-login:%s", testIP)
	wantRefreshKey := fmt.Sprintf("ratelimit:auth-refresh:%s", testIP)

	if wantLoginKey == wantRefreshKey {
		t.Fatalf("login (key=%q) and refresh (key=%q) must use distinct keys", wantLoginKey, wantRefreshKey)
	}
	if wantLoginKey != "ratelimit:auth-login:10.0.0.1" {
		t.Fatalf("login key incorrect: got %q, want %q", wantLoginKey, "ratelimit:auth-login:10.0.0.1")
	}
	if wantRefreshKey != "ratelimit:auth-refresh:10.0.0.1" {
		t.Fatalf("refresh key incorrect: got %q, want %q", wantRefreshKey, "ratelimit:auth-refresh:10.0.0.1")
	}
}

func TestClientIPIgnoresSpoofedForwardingHeadersFromPublicPeer(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "198.51.100.10:443"
	req.Header.Set("X-Forwarded-For", "203.0.113.77")
	req.Header.Set("X-Real-Ip", "203.0.113.88")

	if got := clientIP(req); got != "198.51.100.10" {
		t.Fatalf("clientIP() = %q, want %q", got, "198.51.100.10")
	}
}

func TestClientIPUsesForwardedForFromTrustedProxy(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-For", "203.0.113.77, 10.0.0.2")

	if got := clientIP(req); got != "203.0.113.77" {
		t.Fatalf("clientIP() = %q, want %q", got, "203.0.113.77")
	}
}

func TestClientIPUsesRealIPFromTrustedPrivateProxy(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.10:8080"
	req.Header.Set("X-Real-Ip", "203.0.113.88")

	if got := clientIP(req); got != "203.0.113.88" {
		t.Fatalf("clientIP() = %q, want %q", got, "203.0.113.88")
	}
}

func TestClientIPFallsBackWhenTrustedProxyHeaderIsInvalid(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[::1]:8080"
	req.Header.Set("X-Forwarded-For", "not-an-ip")

	if got := clientIP(req); got != "::1" {
		t.Fatalf("clientIP() = %q, want %q", got, "::1")
	}
}
