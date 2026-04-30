package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/stratahq/backend/internal/audit"
	"github.com/stratahq/backend/internal/config"
	"github.com/stratahq/backend/internal/integrations"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(server.Close)

	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = rdb.Close()
	})
	return rdb
}

func testHandlers() Handlers {
	return Handlers{
		Integrations: integrations.NewHandler(integrations.NewService(nil)),
	}
}

func TestRouterAppliesDistinctRateLimitPrefixesForLoginAndRefresh(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewJSONHandler(&nopWriter{}, &slog.HandlerOptions{}))

	rdb := newTestRedis(t)

	ctx := context.Background()

	existingKeys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	if len(existingKeys) > 0 {
		rdb.Del(ctx, existingKeys...)
	}

	router := NewRouter(cfg, logger, rdb, &silentRecorder{}, testHandlers())

	testIP := "192.0.2.1"

	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	reqLogin.RemoteAddr = testIP + ":12345"
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)

	reqRefresh := httptest.NewRequest("POST", "/api/v1/auth/refresh", nil)
	reqRefresh.RemoteAddr = testIP + ":12345"
	wRefresh := httptest.NewRecorder()
	router.ServeHTTP(wRefresh, reqRefresh)

	keys, _ := rdb.Keys(ctx, "ratelimit:*").Result()

	var loginPrefixUsed, refreshPrefixUsed bool
	for _, k := range keys {
		if strings.HasPrefix(k, "ratelimit:auth-login:") {
			loginPrefixUsed = true
		}
		if strings.HasPrefix(k, "ratelimit:auth-refresh:") {
			refreshPrefixUsed = true
		}
	}

	if !loginPrefixUsed {
		t.Error("login route should use 'ratelimit:auth-login:*' key, but found none")
	}
	if !refreshPrefixUsed {
		t.Error("refresh route should use 'ratelimit:auth-refresh:*' key, but found none")
	}
}

func TestRouterUsesDedicatedRateLimitPrefixesForAIAndWebhooks(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewJSONHandler(&nopWriter{}, &slog.HandlerOptions{}))
	rdb := newTestRedis(t)

	ctx := context.Background()
	existingKeys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	if len(existingKeys) > 0 {
		rdb.Del(ctx, existingKeys...)
	}

	router := NewRouter(cfg, logger, rdb, &silentRecorder{}, testHandlers())
	testIP := "192.0.2.44"

	requests := []struct {
		method string
		path   string
		prefix string
	}{
		{method: "POST", path: "/api/v1/ai/copilot", prefix: "ai-copilot"},
		{method: "POST", path: "/api/v1/billing/webhooks/stripe", prefix: "stripe-webhook"},
		{method: "POST", path: "/api/v1/whatsapp/webhooks", prefix: "twilio-webhook"},
	}

	for _, tc := range requests {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
		req.RemoteAddr = testIP + ":12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	keys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	for _, prefix := range []string{"ai-copilot", "stripe-webhook", "twilio-webhook"} {
		found := false
		for _, key := range keys {
			if strings.HasPrefix(key, "ratelimit:"+prefix+":") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing dedicated rate limit prefix %q in keys %v", prefix, keys)
		}
	}
}

func TestRouterAppliesDedicatedRateLimitPrefixesForPublicAuthEndpoints(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewJSONHandler(&nopWriter{}, &slog.HandlerOptions{}))
	rdb := newTestRedis(t)

	ctx := context.Background()
	existingKeys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	if len(existingKeys) > 0 {
		rdb.Del(ctx, existingKeys...)
	}

	router := NewRouter(cfg, logger, rdb, &silentRecorder{}, testHandlers())
	testIP := "192.0.2.88"

	requests := []struct {
		method string
		path   string
		body   string
		prefix string
	}{
		{method: "POST", path: "/api/v1/auth/register", body: `{}`, prefix: "auth-register"},
		{method: "POST", path: "/api/v1/auth/logout", body: `{}`, prefix: "auth-logout"},
		{method: "POST", path: "/api/v1/auth/forgot-password", body: `{}`, prefix: "auth-forgot-password"},
		{method: "POST", path: "/api/v1/auth/reset-password", body: `{}`, prefix: "auth-reset-password"},
	}

	for _, tc := range requests {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.RemoteAddr = testIP + ":12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	keys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	for _, tc := range requests {
		found := false
		for _, key := range keys {
			if strings.HasPrefix(key, "ratelimit:"+tc.prefix+":") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing dedicated rate limit prefix %q in keys %v", tc.prefix, keys)
		}
	}
}

func TestDistinctRateLimitPrefixesRequiredForLoginVsRefresh(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewJSONHandler(&nopWriter{}, &slog.HandlerOptions{}))

	rdb := newTestRedis(t)

	ctx := context.Background()

	existingKeys, _ := rdb.Keys(ctx, "ratelimit:*").Result()
	if len(existingKeys) > 0 {
		rdb.Del(ctx, existingKeys...)
	}

	router := NewRouter(cfg, logger, rdb, &silentRecorder{}, testHandlers())

	testIP := "192.0.2.1"

	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	reqLogin.RemoteAddr = testIP + ":12345"
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)

	reqRefresh := httptest.NewRequest("POST", "/api/v1/auth/refresh", nil)
	reqRefresh.RemoteAddr = testIP + ":12345"
	wRefresh := httptest.NewRecorder()
	router.ServeHTTP(wRefresh, reqRefresh)

	keys, _ := rdb.Keys(ctx, "ratelimit:*").Result()

	var prefixes []string
	for _, k := range keys {
		parts := strings.Split(k, ":")
		if len(parts) >= 2 && parts[0] == "ratelimit" {
			prefixes = append(prefixes, parts[1])
		}
	}

	hasLogin := false
	hasRefresh := false
	for _, p := range prefixes {
		if p == "auth-login" {
			hasLogin = true
		}
		if p == "auth-refresh" {
			hasRefresh = true
		}
	}

	if !hasLogin {
		t.Error("login endpoint must use 'auth-login' rate limit prefix")
	}
	if !hasRefresh {
		t.Error("refresh endpoint must use 'auth-refresh' rate limit prefix")
	}

	if !hasLogin || !hasRefresh {
		t.Fatalf("FAIL: Both 'auth-login' and 'auth-refresh' prefixes required. Found: %v", prefixes)
	}
}

func TestRouterMountsOpenAPIRoutes(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewJSONHandler(&nopWriter{}, &slog.HandlerOptions{}))
	rdb := newTestRedis(t)

	router := NewRouter(cfg, logger, rdb, &silentRecorder{}, testHandlers())

	req := httptest.NewRequest(http.MethodGet, "/api/open/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("openapi.json: status=%d body=%s", w.Code, w.Body)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("openapi.json content-type: %q", w.Header().Get("Content-Type"))
	}
}

type nopWriter struct{}

func (n *nopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type silentRecorder struct{}

func (s *silentRecorder) Record(ctx context.Context, event audit.Event) error {
	return nil
}
