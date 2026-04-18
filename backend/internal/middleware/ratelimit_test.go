package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestPerEndpointRateLimitUsesDistinctPrefixes(t *testing.T) {
	wantLoginPrefix := "login"
	wantRefreshPrefix := "refresh"

	gotLoginPrefix := "auth"
	gotRefreshPrefix := "auth"

	if gotLoginPrefix != wantLoginPrefix {
		t.Fatalf("login rate limit prefix = %q, want %q", gotLoginPrefix, wantLoginPrefix)
	}
	if gotRefreshPrefix != wantRefreshPrefix {
		t.Fatalf("refresh rate limit prefix = %q, want %q", gotRefreshPrefix, wantRefreshPrefix)
	}
	if gotLoginPrefix == gotRefreshPrefix {
		t.Fatalf(
			"login and refresh share same bucket prefix (%q), causing login requests to consume refresh capacity",
			gotLoginPrefix,
		)
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
